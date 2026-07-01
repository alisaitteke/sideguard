package bootstrap

import (
	"os"
	"path/filepath"
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
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config.yaml missing: %v", err)
	}

	sigPath := filepath.Join(home, ".vibeguard", "signatures", "default.yaml")
	if _, err := os.Stat(sigPath); err != nil {
		t.Fatalf("signatures/default.yaml missing: %v", err)
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
}
