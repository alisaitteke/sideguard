package install_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alisaitteke/sideguard/internal/install"
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

	changed, _, err := install.PatchCursorMCP(path, "/usr/local/bin/sideguard", false)
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
	if fs["command"] != "/usr/local/bin/sideguard" {
		t.Fatalf("unexpected command: %v", fs["command"])
	}
	args := fs["args"].([]any)
	if len(args) < 3 || args[0] != "wrap" || args[1] != "--" || args[2] != "npx" {
		t.Fatalf("unexpected args: %v", args)
	}

	changed2, _, err := install.PatchCursorMCP(path, "/usr/local/bin/sideguard", false)
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

	added, _, err := install.PatchCursorHooks(path, "/bin/sideguard", false)
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

	changed, _, err := install.PatchClaudeMCP(path, "/opt/sideguard", false)
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

	vgHome := filepath.Join(home, ".sideguard", "backups")
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

	backups := filepath.Join(home, ".sideguard", "backups")
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

	changed, _, err := install.PatchCursorMCP(mcpPath, "/bin/sideguard", false)
	if err != nil || changed != 1 {
		t.Fatalf("patch mcp: changed=%d err=%v", changed, err)
	}
	added, _, err := install.PatchCursorHooks(hooksPath, "/bin/sideguard", false)
	if err != nil || added != 2 {
		t.Fatalf("patch hooks: added=%d err=%v", added, err)
	}

	if _, err := install.Uninstall(install.Options{Cursor: true, KeepDaemon: true}); err != nil {
		t.Fatal(err)
	}

	gotMCP, _ := os.ReadFile(mcpPath)
	if !jsonEqual(t, string(gotMCP), initialMCP) {
		t.Fatalf("mcp not restored: %s", gotMCP)
	}
	gotHooks, _ := os.ReadFile(hooksPath)
	if !strings.Contains(string(gotHooks), "keep.sh") {
		t.Fatalf("hooks not restored: %s", gotHooks)
	}
	if strings.Contains(string(gotHooks), "hook shell") {
		t.Fatal("sideguard hooks still present")
	}
}

func TestUnpatchMCPServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	initial := `{"mcpServers":{"fs":{"command":"npx","args":["-y","server"]}}}`
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := install.PatchCursorMCP(path, "/bin/sideguard", false); err != nil {
		t.Fatal(err)
	}
	unwrapped, _, err := install.UnpatchCursorMCP(path, "/bin/sideguard", false)
	if err != nil {
		t.Fatal(err)
	}
	if unwrapped != 1 {
		t.Fatalf("expected 1 unwrapped, got %d", unwrapped)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// JSON formatting may differ; compare structure.
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	fs := doc["mcpServers"].(map[string]any)["fs"].(map[string]any)
	if fs["command"] != "npx" {
		t.Fatalf("command not restored: %v", fs["command"])
	}
}

func TestUnpatchCursorHooksPreservesUserHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	initial := `{"version":1,"hooks":{"beforeReadFile":[{"command":"./keep.sh"}],"beforeShellExecution":[],"beforeMCPExecution":[]}}`
	patched := initial
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := install.PatchCursorHooks(path, "/bin/sideguard", false); err != nil {
		t.Fatal(err)
	}

	removed, _, err := install.UnpatchCursorHooks(path, "/bin/sideguard", false)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Fatalf("expected 2 removed, got %d", removed)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "keep.sh") {
		t.Fatal("user hook removed")
	}
	if strings.Contains(string(raw), "hook shell") {
		t.Fatal("sideguard shell hook still present")
	}
}

func TestUninstallIdempotent(t *testing.T) {
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
	if _, _, err := install.PatchCursorMCP(mcpPath, "/bin/sideguard", false); err != nil {
		t.Fatal(err)
	}

	opts := install.Options{Cursor: true, KeepDaemon: true}
	if _, err := install.Uninstall(opts); err != nil {
		t.Fatal(err)
	}
	afterFirst, _ := os.ReadFile(mcpPath)

	result, err := install.Uninstall(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.FilesChanged) != 0 {
		t.Fatalf("second uninstall changed files: %v", result.FilesChanged)
	}

	afterSecond, _ := os.ReadFile(mcpPath)
	if string(afterFirst) != string(afterSecond) {
		t.Fatal("idempotent uninstall mutated config")
	}
}

func TestRestoreFirstVsRestoreLatest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	orig := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(orig), 0o755); err != nil {
		t.Fatal(err)
	}
	clean := `{"mcpServers":{"a":{"command":"npx"}}}`
	patched := `{"mcpServers":{"a":{"command":"/bin/sideguard","args":["wrap","--","npx"]}}}`
	if err := os.WriteFile(orig, []byte(clean), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := install.CreateBackup([]string{orig}); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(orig, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}

	// Simulate second install backup of already-patched file.
	backupsBase := filepath.Join(home, ".sideguard", "backups")
	entries, err := os.ReadDir(backupsBase)
	if err != nil {
		t.Fatal(err)
	}
	firstSession := filepath.Join(backupsBase, entries[0].Name())
	secondSession := filepath.Join(backupsBase, "20990102-000000")
	if err := os.MkdirAll(secondSession, 0o755); err != nil {
		t.Fatal(err)
	}
	relName := "home/.cursor/mcp.json"
	dst := filepath.Join(secondSession, relName)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`{"timestamp":"20990102-000000","files":{"%s":"%s"}}`, orig, relName)
	if err := os.WriteFile(filepath.Join(secondSession, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = firstSession

	if err := install.RestoreLatest([]string{orig}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(orig)
	if string(raw) != patched {
		t.Fatalf("RestoreLatest should restore newest (patched), got %s", raw)
	}

	if err := os.WriteFile(orig, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := install.RestoreFirst([]string{orig}); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(orig)
	if string(raw) != clean {
		t.Fatalf("RestoreFirst should restore oldest (clean), got %s", raw)
	}
}

func TestUninstallSurgicalAfterDoubleInstallBackup(t *testing.T) {
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

	// First install: backup clean, then patch.
	if _, err := install.CreateBackup([]string{mcpPath, hooksPath}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := install.PatchCursorMCP(mcpPath, "/bin/sideguard", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := install.PatchCursorHooks(hooksPath, "/bin/sideguard", false); err != nil {
		t.Fatal(err)
	}

	// Second install: backup patched state (the bug scenario).
	if _, err := install.CreateBackup([]string{mcpPath, hooksPath}); err != nil {
		t.Fatal(err)
	}
	// Re-patch is idempotent.
	if _, _, err := install.PatchCursorMCP(mcpPath, "/bin/sideguard", false); err != nil {
		t.Fatal(err)
	}

	// Surgical uninstall should clean configs without relying on backup order.
	if _, err := install.Uninstall(install.Options{Cursor: true, KeepDaemon: true}); err != nil {
		t.Fatal(err)
	}

	gotMCP, _ := os.ReadFile(mcpPath)
	if !jsonEqual(t, string(gotMCP), initialMCP) {
		t.Fatalf("mcp not surgically restored: %s", gotMCP)
	}
	gotHooks, _ := os.ReadFile(hooksPath)
	if !strings.Contains(string(gotHooks), "keep.sh") {
		t.Fatal("user hook lost after surgical uninstall")
	}
	if strings.Contains(string(gotHooks), "hook shell") {
		t.Fatal("sideguard hook still present after surgical uninstall")
	}
}

func jsonEqual(t *testing.T, a, b string) bool {
	t.Helper()
	var ja, jb any
	if err := json.Unmarshal([]byte(a), &ja); err != nil {
		t.Fatalf("invalid json a: %v", err)
	}
	if err := json.Unmarshal([]byte(b), &jb); err != nil {
		t.Fatalf("invalid json b: %v", err)
	}
	ab, _ := json.Marshal(ja)
	bb, _ := json.Marshal(jb)
	return string(ab) == string(bb)
}
