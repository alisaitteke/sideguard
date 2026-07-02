package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".vibeguard"), 0o700); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeConfig(t *testing.T, home, content string) {
	t.Helper()
	path := filepath.Join(home, ".vibeguard", "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeWorkspacePolicy(t *testing.T, cwd, content string) {
	t.Helper()
	dir := filepath.Join(cwd, ".vibeguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "policy.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMissing(t *testing.T) {
	setupHome(t)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("expected LLM disabled when config missing")
	}
	if cfg.Provider != "openai" || cfg.Model != "gpt-4o-mini" || cfg.TimeoutMS != 3000 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.Signature != "default" {
		t.Fatalf("expected signature default, got %q", cfg.Signature)
	}
}

func TestLoadValidConfig(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `llm:
  enabled: true
  provider: anthropic
  model: claude-3-5-haiku
  timeout_ms: 5000
  base_url: https://custom.example
  signature: strict
`)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("expected enabled true")
	}
	if cfg.Provider != "anthropic" || cfg.Model != "claude-3-5-haiku" {
		t.Fatalf("unexpected provider/model: %+v", cfg)
	}
	if cfg.TimeoutMS != 5000 || cfg.BaseURL != "https://custom.example" {
		t.Fatalf("unexpected timeout/base_url: %+v", cfg)
	}
	if cfg.Signature != "strict" {
		t.Fatalf("expected signature strict, got %q", cfg.Signature)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, "llm: [not a mapping")

	if _, err := Load(""); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadWorkspaceOverrideDisables(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `llm:
  enabled: true
`)

	ws := t.TempDir()
	writeWorkspacePolicy(t, ws, `llm:
  enabled: false
rules: []
`)

	cfg, err := Load(ws)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("workspace llm.enabled:false should override global true")
	}
}

func TestLoadWorkspaceOverrideEnables(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `llm:
  enabled: false
`)

	ws := t.TempDir()
	writeWorkspacePolicy(t, ws, `llm:
  enabled: true
rules: []
`)

	cfg, err := Load(ws)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("workspace llm.enabled:true should override global false")
	}
}

func TestResolveCredentialsFromFile(t *testing.T) {
	home := setupHome(t)
	path := filepath.Join(home, ".vibeguard", "credentials.yaml")
	content := `openai:
  api_key: file-openai
anthropic:
  api_key: file-anthropic
ollama:
  api_key: ""
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	creds, err := ResolveCredentials()
	if err != nil {
		t.Fatalf("ResolveCredentials() error: %v", err)
	}
	if creds.OpenAI.APIKey != "file-openai" || creds.Anthropic.APIKey != "file-anthropic" {
		t.Fatalf("unexpected creds: %+v", creds)
	}
}

func TestResolveCredentialsEnvOverride(t *testing.T) {
	home := setupHome(t)
	path := filepath.Join(home, ".vibeguard", "credentials.yaml")
	if err := os.WriteFile(path, []byte(`openai:
  api_key: file-key
`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envOpenAIKey, "env-openai")
	t.Setenv(envAnthropicKey, "env-anthropic")

	creds, err := ResolveCredentials()
	if err != nil {
		t.Fatalf("ResolveCredentials() error: %v", err)
	}
	if creds.OpenAI.APIKey != "env-openai" {
		t.Fatalf("env should override file for openai, got %q", creds.OpenAI.APIKey)
	}
	if creds.Anthropic.APIKey != "env-anthropic" {
		t.Fatalf("env should set anthropic, got %q", creds.Anthropic.APIKey)
	}
}

func TestResolveCredentialsMissingFile(t *testing.T) {
	setupHome(t)

	creds, err := ResolveCredentials()
	if err != nil {
		t.Fatalf("ResolveCredentials() error: %v", err)
	}
	if creds.OpenAI.APIKey != "" || creds.Anthropic.APIKey != "" {
		t.Fatalf("expected empty creds, got %+v", creds)
	}
}

func TestEnsureDefaultWritesConfig(t *testing.T) {
	home := setupHome(t)

	path, err := EnsureDefault()
	if err != nil {
		t.Fatalf("EnsureDefault() error: %v", err)
	}
	want := filepath.Join(home, ".vibeguard", "config.yaml")
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
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "history:") || !strings.Contains(content, "retention_days: 30") {
		t.Fatalf("config missing history defaults:\n%s", data)
	}
	if !strings.Contains(content, "update:") || !strings.Contains(content, "check_interval: 6h") {
		t.Fatalf("config missing update defaults:\n%s", data)
	}
}

func TestLoadHistoryDefaults(t *testing.T) {
	setupHome(t)

	cfg, err := LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetentionDays != 30 || cfg.MaxEvents != 50000 {
		t.Fatalf("defaults = %+v", cfg)
	}
}

func TestLoadHistoryFromFile(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `history:
  retention_days: 7
  max_events: 1000
`)

	cfg, err := LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetentionDays != 7 || cfg.MaxEvents != 1000 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadHistoryZeroRetention(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `history:
  retention_days: 0
  max_events: 100
`)

	cfg, err := LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetentionDays != 0 || cfg.MaxEvents != 100 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadUpdateDefaults(t *testing.T) {
	setupHome(t)

	cfg, err := LoadUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled || cfg.CheckInterval != "6h" || cfg.Channel != "stable" {
		t.Fatalf("defaults = %+v", cfg)
	}
}

func TestLoadUpdateFromFile(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `update:
  enabled: false
  check_interval: 12h
  channel: beta
`)

	cfg, err := LoadUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled || cfg.CheckInterval != "12h" || cfg.Channel != "beta" {
		t.Fatalf("cfg = %+v", cfg)
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
}
