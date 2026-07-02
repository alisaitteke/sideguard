package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/install"
)

func TestDetectVibeguardHooksPresent(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"beforeShellExecution": [{"command": "/usr/local/bin/vibeguard hook shell"}],
			"beforeMCPExecution": [{"command": "/usr/local/bin/vibeguard hook mcp"}]
		}
	}`)
	shell, mcp := detectVibeguardHooks("cursor", data)
	if !shell || !mcp {
		t.Fatalf("expected both hooks present, shell=%v mcp=%v", shell, mcp)
	}
}

func TestDetectVibeguardHooksRemoved(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"beforeShellExecution": [{"command": "/bin/echo noop"}]
		}
	}`)
	shell, mcp := detectVibeguardHooks("cursor", data)
	if shell || mcp {
		t.Fatalf("expected hooks missing, shell=%v mcp=%v", shell, mcp)
	}
}

func TestFindUnwrappedStdioServers(t *testing.T) {
	data := []byte(`{
		"mcpServers": {
			"wrapped": {
				"command": "vibeguard",
				"args": ["wrap", "--", "npx", "server"]
			},
			"direct": {
				"command": "npx",
				"args": ["-y", "some-mcp"]
			},
			"remote": {
				"url": "http://127.0.0.1:9000/mcp"
			}
		}
	}`)
	got := findUnwrappedStdioServers(data)
	if len(got) != 1 || got[0] != "direct" {
		t.Fatalf("unwrapped = %v, want [direct]", got)
	}
}

func TestCheckHooksReportsHighWhenRemoved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	if err := os.WriteFile(path, []byte(`{"hooks":{"beforeShellExecution":[]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	findings := checkHooks(install.Target{Client: install.ClientCursor, Kind: install.KindHooks, Path: path})
	var highShell bool
	for _, f := range findings {
		if f.Severity == SeverityHigh && strings.Contains(f.Check, "shell") {
			highShell = true
		}
	}
	if !highShell {
		t.Fatalf("expected HIGH shell finding, got %+v", findings)
	}
}

func TestCheckLLMConfigDisabledNoWarnings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	findings := checkLLMConfig("")
	if len(findings) != 1 || findings[0].Check != "llm_config_missing" {
		t.Fatalf("expected informational OK, got %+v", findings)
	}
	if findings[0].Severity != SeverityOK {
		t.Fatalf("expected OK severity, got %s", findings[0].Severity)
	}
}

func TestCheckLLMConfigEnabledMissingAPIKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`llm:
  enabled: true
  default_provider: my-openai
  providers:
    - id: my-openai
      driver: openai
      model: gpt-4o-mini
      auth_mode: api_key
`), 0o600); err != nil {
		t.Fatal(err)
	}

	findings := checkLLMConfig("")
	var found bool
	for _, f := range findings {
		if f.Check == "llm_enabled_no_credentials" && f.Severity == SeverityWarn {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected llm_enabled_no_credentials WARN, got %+v", findings)
	}
}

func TestCheckLLMConfigCredentialsPermWarn(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`llm:
  enabled: true
  default_provider: my-ollama
  providers:
    - id: my-ollama
      driver: ollama
      model: llama3.2
      auth_mode: api_key
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "credentials.yaml"), []byte(`providers: {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	findings := checkLLMConfig("")
	var found bool
	for _, f := range findings {
		if f.Check == "llm_credentials_perms" && f.Severity == SeverityWarn {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected llm_credentials_perms WARN, got %+v", findings)
	}
}

func TestCheckLLMConfigEnabledOK(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`llm:
  enabled: true
  default_provider: my-openai
  providers:
    - id: my-openai
      driver: openai
      model: gpt-4o-mini
      auth_mode: api_key
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "credentials.yaml"), []byte(`providers:
  my-openai:
    api_key: sk-test
`), 0o600); err != nil {
		t.Fatal(err)
	}

	findings := checkLLMConfig("")
	if len(findings) != 1 || findings[0].Severity != SeverityOK {
		t.Fatalf("expected single OK finding, got %+v", findings)
	}
}

func TestCheckMCPWrapWarnsUnwrapped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	payload := `{"mcpServers":{"fs":{"command":"npx","args":["mcp-server"]}}}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}

	findings := checkMCPWrap(install.Target{Client: install.ClientCursor, Kind: install.KindMCP, Path: path})
	if len(findings) != 1 || findings[0].Severity != SeverityWarn {
		t.Fatalf("expected WARN, got %+v", findings)
	}
	if !strings.Contains(findings[0].Message, "fs") {
		t.Fatalf("expected server name in message: %s", findings[0].Message)
	}
}