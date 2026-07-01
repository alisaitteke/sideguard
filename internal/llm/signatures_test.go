package llm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSignatureFromDir(t *testing.T) {
	dir := filepath.Join("testdata")
	system, err := loadSignatureFromDir(dir, "default")
	if err != nil {
		t.Fatalf("loadSignatureFromDir() error: %v", err)
	}
	if !strings.Contains(system, "Respond with JSON only") {
		t.Fatalf("unexpected system prompt: %q", system)
	}
}

func TestLoadSignatureMissingSystem(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte("name: empty\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSignatureFromDir(dir, "empty"); err == nil {
		t.Fatal("expected error for missing system field")
	}
}

func TestLoadSignatureMissingFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := loadSignatureFromDir(dir, "nope"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadSignatureViaHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sigDir := filepath.Join(home, ".vibeguard", "signatures")
	if err := os.MkdirAll(sigDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sigDir, "custom.yaml"), []byte(`name: custom
system: |
  Custom prompt body
`), 0o600); err != nil {
		t.Fatal(err)
	}

	system, err := LoadSignature("custom")
	if err != nil {
		t.Fatalf("LoadSignature() error: %v", err)
	}
	if system != "Custom prompt body\n" {
		t.Fatalf("got %q", system)
	}
}

func TestEnsureDefaultSignature(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := EnsureDefaultSignature()
	if err != nil {
		t.Fatalf("EnsureDefaultSignature() error: %v", err)
	}
	want := filepath.Join(home, ".vibeguard", "signatures", "default.yaml")
	if path != want {
		t.Fatalf("path %q, want %q", path, want)
	}

	system, err := LoadSignature("default")
	if err != nil {
		t.Fatalf("LoadSignature() error: %v", err)
	}
	if !strings.Contains(system, "destructive patterns") {
		t.Fatalf("unexpected default template: %q", system)
	}
}
