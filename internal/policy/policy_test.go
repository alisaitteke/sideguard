package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsControlPlaneCommand(t *testing.T) {
	allowed := []string{
		"vibeguard pending",
		"vibeguard pending --json",
		"vibeguard ui",
		"vibeguard approve abc-123",
		"vibeguard deny x --reason no",
		"vibeguard status",
		"vibeguard daemon start",
		"vibeguard uninstall",
		"vibeguard doctor",
		"vibeguard policy validate",
		"vibeguard policy init-dev",
		"vibeguard clients reload",
	}
	for _, cmd := range allowed {
		if !IsControlPlaneCommand(cmd) {
			t.Errorf("expected control plane: %q", cmd)
		}
	}

	denied := []string{
		"",
		"vibeguard install",
		"vibeguard wrap -- npx foo",
		"sudo vibeguard pending",
		"echo vibeguard pending",
	}
	for _, cmd := range denied {
		if IsControlPlaneCommand(cmd) {
			t.Errorf("expected not control plane: %q", cmd)
		}
	}
}

func TestDefaultTemplateControlPlaneAllow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(DefaultTemplate), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"vibeguard pending", "vibeguard ui", "vibeguard approve id", "vibeguard daemon status"} {
		res := p.Evaluate(Input{Command: cmd})
		if res.Action != ActionAllow {
			t.Fatalf("cmd %q: got %q, want allow", cmd, res.Action)
		}
	}
}

func TestEvaluatePrecedenceDenyWins(t *testing.T) {
	p, err := FromRules([]Rule{
		{Match: Match{Command: ".*"}, Action: ActionAllow},
		{Match: Match{Command: "curl"}, Action: ActionDeny, Reason: "no curl"},
	})
	if err != nil {
		t.Fatal(err)
	}

	res := p.Evaluate(Input{Command: "curl evil.com"})
	if res.Action != ActionDeny || res.Reason != "no curl" {
		t.Fatalf("got %+v, want deny/no curl", res)
	}
}

func TestEvaluateAskOverAllow(t *testing.T) {
	p, err := FromRules([]Rule{
		{Match: Match{Command: "^git "}, Action: ActionAllow},
		{Match: Match{Command: "^(curl|wget) "}, Action: ActionAsk},
	})
	if err != nil {
		t.Fatal(err)
	}

	res := p.Evaluate(Input{Command: "curl evil.com"})
	if res.Action != ActionAsk {
		t.Fatalf("got %q, want ask", res.Action)
	}

	res = p.Evaluate(Input{Command: "git status"})
	if res.Action != ActionAllow {
		t.Fatalf("got %q, want allow", res.Action)
	}
}

func TestEvaluateMCPToolDeny(t *testing.T) {
	p, err := FromRules([]Rule{
		{Match: Match{MCPTool: ".*delete.*"}, Action: ActionDeny, Reason: "destructive"},
	})
	if err != nil {
		t.Fatal(err)
	}

	res := p.Evaluate(Input{ToolName: "memory_delete", Command: "mcp:memory_delete"})
	if res.Action != ActionDeny {
		t.Fatalf("got %+v, want deny", res)
	}
}

func TestEvaluateDefaultAsk(t *testing.T) {
	p, err := FromRules(nil)
	if err != nil {
		t.Fatal(err)
	}
	res := p.Evaluate(Input{Command: "rm -rf /"})
	if res.Action != ActionAsk {
		t.Fatalf("got %q, want ask", res.Action)
	}
}

func TestLoadInvalidRegex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(`rules:
  - match: { command: "[invalid" }
    action: allow
`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected compile error")
	}
}

func TestLoadGlobalAndWorkspaceMerge(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()

	globalDir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(globalDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "policy.yaml"), []byte(`rules:
  - match: { command: "^git status" }
    action: allow
`), 0o600); err != nil {
		t.Fatal(err)
	}

	wsDir := filepath.Join(cwd, ".vibeguard")
	if err := os.MkdirAll(wsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "policy.yaml"), []byte(`rules:
  - match: { command: "^npm test" }
    action: allow
`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)

	p, err := Load(cwd)
	if err != nil {
		t.Fatal(err)
	}

	if res := p.Evaluate(Input{Command: "git status"}); res.Action != ActionAllow {
		t.Fatalf("global rule: got %q", res.Action)
	}
	if res := p.Evaluate(Input{Command: "npm test"}); res.Action != ActionAllow {
		t.Fatalf("workspace rule: got %q", res.Action)
	}
	if res := p.Evaluate(Input{Command: "curl x"}); res.Action != ActionAsk {
		t.Fatalf("default: got %q", res.Action)
	}
}

func TestEvaluateFailClosedOnLoadError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	globalDir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(globalDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "policy.yaml"), []byte(`rules:
  - match: { command: "(" }
    action: allow
`), 0o600); err != nil {
		t.Fatal(err)
	}

	res := Evaluate("/tmp", Input{Command: "echo hi"})
	if res.Action != ActionDeny || res.LoadError == nil {
		t.Fatalf("expected fail-closed deny, got %+v", res)
	}
}

func TestAppendAllowRule(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := AppendAllowRule(Match{Command: "^echo test$"}, "always allow from approve"); err != nil {
		t.Fatal(err)
	}

	p, err := Load("/tmp")
	if err != nil {
		t.Fatal(err)
	}
	res := p.Evaluate(Input{Command: "echo test"})
	if res.Action != ActionAllow {
		t.Fatalf("got %q, want allow", res.Action)
	}
}
