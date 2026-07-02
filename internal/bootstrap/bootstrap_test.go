package bootstrap

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
	return home
}

func TestEnsureDefaultsWritesBothFiles(t *testing.T) {
	home := setupHome(t)

	if err := EnsureDefaults(); err != nil {
		t.Fatalf("EnsureDefaults() error: %v", err)
	}

	configPath := filepath.Join(home, ".vibeguard", "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config.yaml missing: %v", err)
	}
	if !strings.Contains(string(configData), "update:") {
		t.Fatalf("config missing update block:\n%s", configData)
	}

	sigPath := filepath.Join(home, ".vibeguard", "signatures", "default.yaml")
	if _, err := os.Stat(sigPath); err != nil {
		t.Fatalf("signatures/default.yaml missing: %v", err)
	}

	rulesDir := filepath.Join(home, ".vibeguard", "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		t.Fatalf("rules dir missing: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected embedded detect rules on disk")
	}
}

func TestEnsureDefaultsIdempotent(t *testing.T) {
	setupHome(t)

	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	data1, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".vibeguard", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	data2, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".vibeguard", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data1) != string(data2) {
		t.Fatal("EnsureDefaults modified existing config.yaml")
	}

	rulesDir := filepath.Join(os.Getenv("HOME"), ".vibeguard", "rules")
	custom := filepath.Join(rulesDir, "custom.yaml")
	customContent := []byte("rules: []\n")
	if err := os.WriteFile(custom, customContent, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(custom)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(customContent) {
		t.Fatal("EnsureDefaults overwrote user rules file")
	}
}
