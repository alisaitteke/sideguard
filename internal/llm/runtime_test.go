package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T, home string, content string) {
	t.Helper()
	dir := filepath.Join(home, ".sideguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestEnabledDefaultOff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if Enabled("/tmp") {
		t.Fatal("expected LLM disabled when config missing")
	}
}

func TestEnabledFromConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeTestConfig(t, home, `llm:
  enabled: true
  default_provider: my-openai
  providers:
    - id: my-openai
      driver: openai
      model: gpt-4o-mini
      auth_mode: api_key
`)

	if !Enabled("/tmp") {
		t.Fatal("expected LLM enabled from config")
	}
}

func TestEnabledWorkspaceOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeTestConfig(t, home, `llm:
  enabled: true
`)

	cwd := t.TempDir()
	workspaceDir := filepath.Join(cwd, ".sideguard")
	if err := os.MkdirAll(workspaceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	policy := `llm:
  enabled: false
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "policy.yaml"), []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}

	if Enabled(cwd) {
		t.Fatal("workspace llm.enabled:false should disable LLM")
	}
}

func TestClassifierForDisabledReturnsNil(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	home := t.TempDir()
	t.Setenv("HOME", home)

	clf, err := ClassifierFor("/tmp")
	if err != nil {
		t.Fatalf("ClassifierFor() error: %v", err)
	}
	if clf != nil {
		t.Fatal("expected nil classifier when LLM disabled")
	}
}

func TestClassifierForEnabledWithoutSignatureFails(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	home := t.TempDir()
	t.Setenv("HOME", home)
	writeTestConfig(t, home, `llm:
  enabled: true
  default_provider: my-openai
  providers:
    - id: my-openai
      driver: openai
      model: gpt-4o-mini
      auth_mode: api_key
`)

	clf, err := ClassifierFor("/tmp")
	if err == nil {
		t.Fatal("expected error when signature missing")
	}
	if clf != nil {
		t.Fatal("expected nil classifier on init failure")
	}
}
