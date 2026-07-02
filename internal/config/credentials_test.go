package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveProviderCredentialsFromFile(t *testing.T) {
	home := setupHome(t)
	path := filepath.Join(home, ".vibeguard", "credentials.yaml")
	content := `providers:
  my-openai:
    api_key: file-openai
  my-anthropic:
    api_key: file-anthropic
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	creds, err := ResolveProviderCredentials()
	if err != nil {
		t.Fatalf("ResolveProviderCredentials() error: %v", err)
	}
	if creds["my-openai"].APIKey != "file-openai" || creds["my-anthropic"].APIKey != "file-anthropic" {
		t.Fatalf("unexpected creds: %+v", creds)
	}
}

func TestResolveProviderCredentialsEnvOverride(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `llm:
  providers:
    - id: my-openai
      driver: openai
      model: gpt-4o-mini
      auth_mode: api_key
    - id: my-anthropic
      driver: anthropic
      model: claude
      auth_mode: api_key
`)
	path := filepath.Join(home, ".vibeguard", "credentials.yaml")
	if err := os.WriteFile(path, []byte(`providers:
  my-openai:
    api_key: file-key
`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envOpenAIKey, "env-openai")
	t.Setenv(envAnthropicKey, "env-anthropic")

	creds, err := ResolveProviderCredentials()
	if err != nil {
		t.Fatalf("ResolveProviderCredentials() error: %v", err)
	}
	if creds["my-openai"].APIKey != "env-openai" {
		t.Fatalf("env should override file for openai driver, got %q", creds["my-openai"].APIKey)
	}
	if creds["my-anthropic"].APIKey != "env-anthropic" {
		t.Fatalf("env should set anthropic driver, got %q", creds["my-anthropic"].APIKey)
	}
}

func TestResolveProviderCredentialsMissingFile(t *testing.T) {
	setupHome(t)

	creds, err := ResolveProviderCredentials()
	if err != nil {
		t.Fatalf("ResolveProviderCredentials() error: %v", err)
	}
	if len(creds) != 0 {
		t.Fatalf("expected empty creds, got %+v", creds)
	}
}

func TestSetProviderKeyWrites0600(t *testing.T) {
	home := setupHome(t)

	if err := SetProviderKey("my-openai", "sk-test-secret-key"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".vibeguard", "credentials.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected mode 0600, got %o", info.Mode().Perm())
	}
	creds, err := ResolveProviderCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if creds["my-openai"].APIKey != "sk-test-secret-key" {
		t.Fatalf("unexpected key: %q", creds["my-openai"].APIKey)
	}
}

func TestProviderStatusMasked(t *testing.T) {
	setupHome(t)
	if err := SetProviderKey("p1", "sk-abcdefghijklmnop"); err != nil {
		t.Fatal(err)
	}
	masked, configured, err := ProviderStatus("p1")
	if err != nil {
		t.Fatal(err)
	}
	if !configured {
		t.Fatal("expected configured true")
	}
	if !strings.Contains(masked, "…") || strings.Contains(masked, "klmnop") {
		t.Fatalf("expected masked key, got %q", masked)
	}
	if strings.HasSuffix(masked, "mnop") {
		// last 4 chars visible per mask format
	} else if !strings.HasSuffix(masked, "mnop") {
		t.Fatalf("expected last 4 chars visible, got %q", masked)
	}
}

func TestProviderStatusMissing(t *testing.T) {
	setupHome(t)
	masked, configured, err := ProviderStatus("missing")
	if err != nil {
		t.Fatal(err)
	}
	if configured || masked != "" {
		t.Fatalf("expected unconfigured, got masked=%q configured=%v", masked, configured)
	}
}

func TestEnsureCredentialsDefaultMode0600(t *testing.T) {
	home := setupHome(t)

	path, err := EnsureCredentialsDefault()
	if err != nil {
		t.Fatalf("EnsureCredentialsDefault() error: %v", err)
	}
	want := filepath.Join(home, ".vibeguard", "credentials.yaml")
	if path != want {
		t.Fatalf("path %q, want %q", path, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected mode 0600, got %o", info.Mode().Perm())
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "providers: {}") {
		t.Fatalf("unexpected template: %s", content)
	}
}

func TestMaskAPIKey(t *testing.T) {
	got := maskAPIKey("sk-abcdefghijklmnop")
	want := "sk-abcd…mnop"
	if got != want {
		t.Fatalf("maskAPIKey() = %q, want %q", got, want)
	}
}
