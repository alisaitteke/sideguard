package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alisaitteke/sideguard/internal/paths"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".sideguard"), 0o700); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeConfig(t *testing.T, home, content string) {
	t.Helper()
	path := filepath.Join(home, ".sideguard", "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeWorkspacePolicy(t *testing.T, cwd, content string) {
	t.Helper()
	dir := filepath.Join(cwd, ".sideguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "policy.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLLMSettingsMissing(t *testing.T) {
	setupHome(t)

	cfg, err := LoadLLMSettings("")
	if err != nil {
		t.Fatalf("LoadLLMSettings() error: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("expected LLM disabled when config missing")
	}
	if cfg.TimeoutMS != 3000 {
		t.Fatalf("unexpected timeout: %d", cfg.TimeoutMS)
	}
	if cfg.Analysis.Signature != "analysis" {
		t.Fatalf("expected analysis signature default, got %q", cfg.Analysis.Signature)
	}
	if len(cfg.Providers) != 0 {
		t.Fatalf("expected empty providers, got %+v", cfg.Providers)
	}
}

func TestLoadLLMSettingsValidConfig(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `llm:
  enabled: true
  default_provider: my-anthropic
  timeout_ms: 5000
  providers:
    - id: my-openai
      driver: openai
      model: gpt-4o-mini
      base_url: ""
      auth_mode: api_key
    - id: my-anthropic
      driver: anthropic
      model: claude-3-5-sonnet-latest
      auth_mode: api_key
  analysis:
    signature: analysis
    provider: my-openai
`)

	cfg, err := LoadLLMSettings("")
	if err != nil {
		t.Fatalf("LoadLLMSettings() error: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("expected enabled true")
	}
	if cfg.DefaultProvider != "my-anthropic" {
		t.Fatalf("default_provider = %q", cfg.DefaultProvider)
	}
	if len(cfg.Providers) != 2 {
		t.Fatalf("providers = %+v", cfg.Providers)
	}
	if cfg.Providers[0].ID != "my-openai" || cfg.Providers[1].Driver != "anthropic" {
		t.Fatalf("unexpected providers: %+v", cfg.Providers)
	}
	if cfg.TimeoutMS != 5000 {
		t.Fatalf("timeout = %d", cfg.TimeoutMS)
	}
	if cfg.Analysis.Provider != "my-openai" {
		t.Fatalf("analysis.provider = %q", cfg.Analysis.Provider)
	}
}

func TestLoadLLMSettingsInvalidYAML(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, "llm: [not a mapping")

	if _, err := LoadLLMSettings(""); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadLLMSettingsWorkspaceOverrideDisables(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `llm:
  enabled: true
`)

	ws := t.TempDir()
	writeWorkspacePolicy(t, ws, `llm:
  enabled: false
rules: []
`)

	cfg, err := LoadLLMSettings(ws)
	if err != nil {
		t.Fatalf("LoadLLMSettings() error: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("workspace llm.enabled:false should override global true")
	}
}

func TestLoadLLMSettingsWorkspaceOverrideEnables(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `llm:
  enabled: false
`)

	ws := t.TempDir()
	writeWorkspacePolicy(t, ws, `llm:
  enabled: true
rules: []
`)

	cfg, err := LoadLLMSettings(ws)
	if err != nil {
		t.Fatalf("LoadLLMSettings() error: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("workspace llm.enabled:true should override global false")
	}
}

func TestEnsureDefaultWritesConfig(t *testing.T) {
	home := setupHome(t)

	path, err := EnsureDefault()
	if err != nil {
		t.Fatalf("EnsureDefault() error: %v", err)
	}
	want := filepath.Join(home, ".sideguard", "config.yaml")
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
	if !strings.Contains(content, "providers: []") {
		t.Fatalf("config missing providers block:\n%s", data)
	}
	if !strings.Contains(content, "analysis:") || !strings.Contains(content, "signature: analysis") {
		t.Fatalf("config missing analysis defaults:\n%s", data)
	}
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

func TestSaveProvidersRoundTrip(t *testing.T) {
	setupHome(t)

	settings := LLMSettings{
		Enabled:         true,
		DefaultProvider: "my-openai",
		TimeoutMS:       4000,
		Providers: []ProviderInstance{
			{ID: "my-openai", Driver: "openai", Model: "gpt-4o-mini", AuthMode: "api_key"},
		},
		Analysis: AnalysisSettings{Signature: "analysis", Provider: ""},
	}
	if err := SaveProviders(settings); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadLLMSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultProvider != "my-openai" || len(loaded.Providers) != 1 {
		t.Fatalf("loaded = %+v", loaded)
	}
	if loaded.Providers[0].Model != "gpt-4o-mini" {
		t.Fatalf("provider = %+v", loaded.Providers[0])
	}

	path, err := paths.ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("expected config mode 0600, got %o", st.Mode().Perm())
	}
}

func TestSaveProvidersPreservesHistoryUpdate(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, `llm:
  enabled: false
history:
  retention_days: 14
  max_events: 2000
update:
  enabled: false
  channel: beta
`)

	settings := LLMSettings{
		Enabled: true,
		Providers: []ProviderInstance{
			{ID: "p1", Driver: "ollama", Model: "llama3", AuthMode: "api_key"},
		},
		Analysis: AnalysisSettings{Signature: "analysis"},
	}
	if err := SaveProviders(settings); err != nil {
		t.Fatal(err)
	}

	hist, err := LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if hist.RetentionDays != 14 || hist.MaxEvents != 2000 {
		t.Fatalf("history not preserved: %+v", hist)
	}
	upd, err := LoadUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if upd.Enabled || upd.Channel != "beta" {
		t.Fatalf("update not preserved: %+v", upd)
	}
}

func TestSaveProvidersDuplicateIDRejected(t *testing.T) {
	setupHome(t)

	settings := LLMSettings{
		Providers: []ProviderInstance{
			{ID: "dup", Driver: "openai", Model: "m", AuthMode: "api_key"},
			{ID: "dup", Driver: "anthropic", Model: "m", AuthMode: "api_key"},
		},
	}
	if err := SaveProviders(settings); err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestSaveProvidersInvalidDefaultProvider(t *testing.T) {
	setupHome(t)

	settings := LLMSettings{
		DefaultProvider: "missing",
		Providers: []ProviderInstance{
			{ID: "p1", Driver: "openai", Model: "m", AuthMode: "api_key"},
		},
	}
	if err := SaveProviders(settings); err == nil {
		t.Fatal("expected default_provider validation error")
	}
}

func TestSetDefaultProvider(t *testing.T) {
	setupHome(t)
	settings := LLMSettings{
		Providers: []ProviderInstance{
			{ID: "a", Driver: "openai", Model: "m", AuthMode: "api_key"},
			{ID: "b", Driver: "anthropic", Model: "m", AuthMode: "api_key"},
		},
		Analysis: AnalysisSettings{Signature: "analysis"},
	}
	if err := SaveProviders(settings); err != nil {
		t.Fatal(err)
	}
	if err := SetDefaultProvider("b"); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadLLMSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultProvider != "b" {
		t.Fatalf("default = %q", loaded.DefaultProvider)
	}
}

func TestRemoveProvider(t *testing.T) {
	setupHome(t)
	settings := LLMSettings{
		DefaultProvider: "keep",
		Providers: []ProviderInstance{
			{ID: "keep", Driver: "openai", Model: "m", AuthMode: "api_key"},
			{ID: "drop", Driver: "anthropic", Model: "m", AuthMode: "api_key"},
		},
		Analysis: AnalysisSettings{Signature: "analysis"},
	}
	if err := SaveProviders(settings); err != nil {
		t.Fatal(err)
	}
	if err := SetProviderKey("drop", "secret"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveProvider("drop"); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadLLMSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Providers) != 1 || loaded.Providers[0].ID != "keep" {
		t.Fatalf("providers = %+v", loaded.Providers)
	}
	creds, err := ResolveProviderCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := creds["drop"]; ok {
		t.Fatal("credentials for removed provider should be deleted")
	}
}

func TestLoadProviders(t *testing.T) {
	setupHome(t)
	settings := LLMSettings{
		Providers: []ProviderInstance{
			{ID: "x", Driver: "ollama", Model: "m", AuthMode: "api_key"},
		},
		Analysis: AnalysisSettings{Signature: "analysis"},
	}
	if err := SaveProviders(settings); err != nil {
		t.Fatal(err)
	}
	providers, err := LoadProviders()
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 1 || providers[0].ID != "x" {
		t.Fatalf("providers = %+v", providers)
	}
}
