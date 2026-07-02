//go:build darwin

package darwin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/config"
)

func setupTraySettingsHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".vibeguard"), 0o700); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeTrayConfig(t *testing.T, home, content string) {
	t.Helper()
	path := filepath.Join(home, ".vibeguard", "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSettingsSnapshotEmpty(t *testing.T) {
	setupTraySettingsHome(t)

	snap, err := LoadSettingsSnapshot()
	if err != nil {
		t.Fatalf("LoadSettingsSnapshot() error: %v", err)
	}
	if len(snap.Drivers) == 0 {
		t.Fatal("expected registered drivers in snapshot")
	}
	if len(snap.Providers) != 0 {
		t.Fatalf("providers = %d, want 0", len(snap.Providers))
	}
}

func TestLoadSettingsSnapshotMaskedKey(t *testing.T) {
	home := setupTraySettingsHome(t)
	writeTrayConfig(t, home, `llm:
  enabled: true
  default_provider: p1
  providers:
    - id: p1
      driver: openai
      model: gpt-4o-mini
      auth_mode: api_key
`)
	if err := config.SetProviderKey("p1", "sk-abcdefghijklmnop"); err != nil {
		t.Fatal(err)
	}

	snap, err := LoadSettingsSnapshot()
	if err != nil {
		t.Fatalf("LoadSettingsSnapshot() error: %v", err)
	}
	if len(snap.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(snap.Providers))
	}
	row := snap.Providers[0]
	if !row.KeyConfigured {
		t.Fatal("expected key_configured true")
	}
	if row.APIKey == "" || row.APIKey == "sk-abcdefghijklmnop" {
		t.Fatalf("expected masked key, got %q", row.APIKey)
	}
	if !row.IsDefault {
		t.Fatal("expected is_default true")
	}
}

func TestSaveSettingsFromJSONRoundTrip(t *testing.T) {
	home := setupTraySettingsHome(t)
	writeTrayConfig(t, home, `llm:
  enabled: true
`)

	payload, err := json.Marshal(settingsSavePayload{
		Providers: []ProviderRowJSON{
			{
				ID:        "my-openai",
				Driver:    "openai",
				Model:     "gpt-4o-mini",
				APIKey:    "sk-new-secret-key-value",
				IsDefault: true,
			},
		},
		DefaultProvider: "my-openai",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := SaveSettingsFromJSON(string(payload)); err != nil {
		t.Fatalf("SaveSettingsFromJSON() error: %v", err)
	}

	loaded, err := config.LoadLLMSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultProvider != "my-openai" {
		t.Fatalf("default_provider = %q", loaded.DefaultProvider)
	}
	if len(loaded.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(loaded.Providers))
	}
	if loaded.Providers[0].Model != "gpt-4o-mini" {
		t.Fatalf("model = %q", loaded.Providers[0].Model)
	}

	creds, err := config.ResolveProviderCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if creds["my-openai"].APIKey != "sk-new-secret-key-value" {
		t.Fatalf("api key = %q", creds["my-openai"].APIKey)
	}
}

func TestSaveSettingsPreservesKeyWhenMasked(t *testing.T) {
	home := setupTraySettingsHome(t)
	writeTrayConfig(t, home, `llm:
  enabled: true
  default_provider: p1
  providers:
    - id: p1
      driver: openai
      model: gpt-4o-mini
      auth_mode: api_key
`)
	if err := config.SetProviderKey("p1", "sk-keep-this-secret"); err != nil {
		t.Fatal(err)
	}

	snap, err := LoadSettingsSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	snap.Providers[0].Model = "gpt-4o"
	payload, err := json.Marshal(settingsSavePayload{
		Providers:       snap.Providers,
		DefaultProvider: "p1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := SaveSettingsFromJSON(string(payload)); err != nil {
		t.Fatalf("SaveSettingsFromJSON() error: %v", err)
	}

	creds, err := config.ResolveProviderCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if creds["p1"].APIKey != "sk-keep-this-secret" {
		t.Fatalf("api key = %q, want preserved", creds["p1"].APIKey)
	}

	loaded, err := config.LoadLLMSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Providers[0].Model != "gpt-4o" {
		t.Fatalf("model = %q", loaded.Providers[0].Model)
	}
}

func TestSaveSettingsDuplicateID(t *testing.T) {
	setupTraySettingsHome(t)

	payload, err := json.Marshal(settingsSavePayload{
		Providers: []ProviderRowJSON{
			{ID: "dup", Driver: "openai", Model: "m"},
			{ID: "dup", Driver: "anthropic", Model: "m"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := SaveSettingsFromJSON(string(payload)); err == nil {
		t.Fatal("expected duplicate id error")
	}
}
