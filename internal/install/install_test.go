package install_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/install"
)

func TestWrapMCPServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	initial := `{
  "mcpServers": {
    "fs": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    }
  }
}`
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := install.PatchCursorMCP(path, "/usr/local/bin/vibeguard", false)
	if err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Fatalf("expected 1 wrapped server, got %d", changed)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	serversMap := doc["mcpServers"].(map[string]any)
	fs := serversMap["fs"].(map[string]any)
	if fs["command"] != "/usr/local/bin/vibeguard" {
		t.Fatalf("unexpected command: %v", fs["command"])
	}
	args := fs["args"].([]any)
	if len(args) < 3 || args[0] != "wrap" || args[1] != "--" || args[2] != "npx" {
		t.Fatalf("unexpected args: %v", args)
	}

	changed2, _, err := install.PatchCursorMCP(path, "/usr/local/bin/vibeguard", false)
	if err != nil {
		t.Fatal(err)
	}
	if changed2 != 0 {
		t.Fatalf("expected no double wrap, got %d", changed2)
	}
}

func TestMergeCursorHooksPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	initial := `{
  "version": 1,
  "hooks": {
    "beforeReadFile": [{ "command": "./my-hook.sh" }],
    "beforeShellExecution": []
  }
}`
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	added, _, err := install.PatchCursorHooks(path, "/bin/vibeguard", false)
	if err != nil {
		t.Fatal(err)
	}
	if added != 2 {
		t.Fatalf("expected 2 hook entries added, got %d", added)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "my-hook.sh") {
		t.Fatal("existing hook was removed")
	}
	if !strings.Contains(string(raw), "hook shell") {
		t.Fatal("shell hook missing")
	}
	if !strings.Contains(string(raw), "hook mcp") {
		t.Fatal("mcp hook missing")
	}
}

func TestPatchClaudeMCPPreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	initial := `{
  "theme": "dark",
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  }
}`
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := install.PatchClaudeMCP(path, "/opt/vibeguard", false)
	if err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Fatalf("expected 1 wrap, got %d", changed)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"theme": "dark"`) {
		t.Fatal("lost unrelated claude.json keys")
	}
}

func TestBackupAndRestore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	vgHome := filepath.Join(home, ".vibeguard", "backups")
	if err := os.MkdirAll(vgHome, 0o755); err != nil {
		t.Fatal(err)
	}

	orig := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(orig), 0o755); err != nil {
		t.Fatal(err)
	}
	before := `{"mcpServers":{}}`
	if err := os.WriteFile(orig, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}

	session, err := install.CreateBackup([]string{orig})
	if err != nil {
		t.Fatal(err)
	}
	if session.Dir == "" {
		t.Fatal("expected backup dir")
	}

	after := `{"mcpServers":{"x":{"command":"npx"}}}`
	if err := os.WriteFile(orig, []byte(after), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := install.RestoreLatest([]string{orig}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(orig)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != before {
		t.Fatalf("restore mismatch:\n%s", raw)
	}
}

func TestInstallDryRunNoWrites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cursorDir := filepath.Join(home, ".cursor")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mcpPath := filepath.Join(cursorDir, "mcp.json")
	initial := `{"mcpServers":{"a":{"command":"npx","args":["-y","server"]}}}`
	if err := os.WriteFile(mcpPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := install.Run(install.Options{Cursor: true, DryRun: true, Cwd: home})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != initial {
		t.Fatal("dry-run mutated mcp.json")
	}

	backups := filepath.Join(home, ".vibeguard", "backups")
	if _, err := os.Stat(backups); err == nil {
		entries, _ := os.ReadDir(backups)
		if len(entries) > 0 {
			t.Fatal("dry-run created backups")
		}
	}
}

func TestInstallUninstallRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cursorDir := filepath.Join(home, ".cursor")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mcpPath := filepath.Join(cursorDir, "mcp.json")
	hooksPath := filepath.Join(cursorDir, "hooks.json")
	initialMCP := `{"mcpServers":{"a":{"command":"npx","args":["-y","server"]}}}`
	initialHooks := `{"version":1,"hooks":{"beforeReadFile":[{"command":"./keep.sh"}]}}`
	if err := os.WriteFile(mcpPath, []byte(initialMCP), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hooksPath, []byte(initialHooks), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := install.CreateBackup([]string{mcpPath, hooksPath}); err != nil {
		t.Fatal(err)
	}

	changed, _, err := install.PatchCursorMCP(mcpPath, "/bin/vibeguard", false)
	if err != nil || changed != 1 {
		t.Fatalf("patch mcp: changed=%d err=%v", changed, err)
	}
	added, _, err := install.PatchCursorHooks(hooksPath, "/bin/vibeguard", false)
	if err != nil || added != 2 {
		t.Fatalf("patch hooks: added=%d err=%v", added, err)
	}

	if err := install.Uninstall(install.Options{Cursor: true}); err != nil {
		t.Fatal(err)
	}

	gotMCP, _ := os.ReadFile(mcpPath)
	if string(gotMCP) != initialMCP {
		t.Fatalf("mcp not restored: %s", gotMCP)
	}
	gotHooks, _ := os.ReadFile(hooksPath)
	if string(gotHooks) != initialHooks {
		t.Fatalf("hooks not restored: %s", gotHooks)
	}
}
