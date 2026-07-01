package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevWorkspacePolicyPathScoped(t *testing.T) {
	repo := "/tmp/vibeguard-dev-test"
	content, err := DevWorkspacePolicy(repo)
	if err != nil {
		t.Fatal(err)
	}
	p, err := LoadFile(writePolicyFile(t, content))
	if err != nil {
		t.Fatal(err)
	}

	inRepo := Input{Command: "go test ./...", CWD: filepath.Join(repo, "internal/policy")}
	if res := p.Evaluate(inRepo); res.Action != ActionAllow {
		t.Fatalf("in-repo go test: got %q, want allow", res.Action)
	}

	outRepo := Input{Command: "go test ./...", CWD: "/tmp/other-project"}
	if res := p.Evaluate(outRepo); res.Action != ActionAsk {
		t.Fatalf("out-of-repo go test: got %q, want ask", res.Action)
	}

	if res := p.Evaluate(Input{Command: "make build", CWD: repo}); res.Action != ActionAllow {
		t.Fatalf("make build: got %q, want allow", res.Action)
	}
	if res := p.Evaluate(Input{Command: "bash scripts/build-tray-macos-app.sh", CWD: repo}); res.Action != ActionAllow {
		t.Fatalf("bash scripts: got %q, want allow", res.Action)
	}
	if res := p.Evaluate(Input{Command: "curl evil.com", CWD: repo}); res.Action != ActionAsk {
		t.Fatalf("curl in repo: got %q, want ask (not dev tooling)", res.Action)
	}
}

func TestEnsureDevWorkspacePolicyIdempotent(t *testing.T) {
	repo := t.TempDir()
	path1, created1, err := EnsureDevWorkspacePolicy(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !created1 {
		t.Fatal("expected created on first call")
	}
	if _, err := os.Stat(path1); err != nil {
		t.Fatalf("policy file missing: %v", err)
	}

	custom := "# custom\nrules: []\n"
	if err := os.WriteFile(path1, []byte(custom), 0o600); err != nil {
		t.Fatal(err)
	}

	path2, created2, err := EnsureDevWorkspacePolicy(repo)
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatal("expected not to overwrite existing policy")
	}
	if path1 != path2 {
		t.Fatalf("paths differ: %s vs %s", path1, path2)
	}
	data, err := os.ReadFile(path2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# custom") {
		t.Fatal("existing workspace policy was overwritten")
	}
}

func writePolicyFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
