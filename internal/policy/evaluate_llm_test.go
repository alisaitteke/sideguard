package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type mockClassifier struct {
	calls  int
	result Result
}

func (m *mockClassifier) Classify(ctx context.Context, input Input, yamlReason string) Result {
	m.calls++
	return m.result
}

func TestEvaluateWithLLMYAMLAllowSkipsLLM(t *testing.T) {
	home := t.TempDir()
	vibeguardDir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(vibeguardDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vibeguardDir, "policy.yaml"), []byte("rules:\n  - match: { command: \"^git status\" }\n    action: allow\n    reason: safe read\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	mock := &mockClassifier{result: Result{Action: ActionDeny, Reason: "llm should not run"}}
	res := EvaluateWithLLM(context.Background(), "/tmp", Input{Command: "git status", CWD: "/tmp"}, mock, true)
	if res.Action != ActionAllow {
		t.Fatalf("got %+v, want allow", res)
	}
	if mock.calls != 0 {
		t.Fatalf("classifier called %d times, want 0 on YAML allow", mock.calls)
	}
}

func TestEvaluateWithLLMYAMLDenySkipsLLM(t *testing.T) {
	home := t.TempDir()
	vibeguardDir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(vibeguardDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `rules:
  - match: { command: "curl" }
    action: deny
    reason: blocked curl
`
	if err := os.WriteFile(filepath.Join(vibeguardDir, "policy.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	mock := &mockClassifier{result: Result{Action: ActionAllow, Reason: "llm should not run"}}
	res := EvaluateWithLLM(context.Background(), "/tmp", Input{Command: "curl evil.com", CWD: "/tmp"}, mock, true)
	if res.Action != ActionDeny {
		t.Fatalf("got %+v, want deny", res)
	}
	if mock.calls != 0 {
		t.Fatalf("classifier called %d times, want 0 on YAML deny", mock.calls)
	}
}

func TestEvaluateWithLLMAskEnabledInvokesLLM(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mock := &mockClassifier{result: Result{Action: ActionAllow, Reason: "llm approved"}}
	res := EvaluateWithLLM(context.Background(), "/tmp", Input{Command: "unknown-cmd", CWD: "/tmp"}, mock, true)
	if res.Action != ActionAllow || res.Reason != "llm approved" {
		t.Fatalf("got %+v, want llm allow", res)
	}
	if mock.calls != 1 {
		t.Fatalf("classifier called %d times, want 1", mock.calls)
	}
}

func TestEvaluateWithLLMAskDisabledSkipsLLM(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mock := &mockClassifier{result: Result{Action: ActionAllow, Reason: "llm should not run"}}
	res := EvaluateWithLLM(context.Background(), "/tmp", Input{Command: "unknown-cmd", CWD: "/tmp"}, mock, false)
	if res.Action != ActionAsk {
		t.Fatalf("got %+v, want ask", res)
	}
	if mock.calls != 0 {
		t.Fatalf("classifier called %d times, want 0 when LLM disabled", mock.calls)
	}
}

func TestEvaluateWithLLMNilClassifierSkipsLLM(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	res := EvaluateWithLLM(context.Background(), "/tmp", Input{Command: "unknown-cmd", CWD: "/tmp"}, nil, true)
	if res.Action != ActionAsk {
		t.Fatalf("got %+v, want ask when classifier nil", res)
	}
}

func TestEvaluateWithLLMLoadErrorSkipsLLM(t *testing.T) {
	home := t.TempDir()
	vibeguardDir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(vibeguardDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vibeguardDir, "policy.yaml"), []byte("rules:\n  - match: { command: \"[\" }\n    action: allow\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	mock := &mockClassifier{result: Result{Action: ActionAllow}}
	res := EvaluateWithLLM(context.Background(), "/tmp", Input{Command: "echo hi", CWD: "/tmp"}, mock, true)
	if res.Action != ActionDeny {
		t.Fatalf("got %+v, want fail-closed deny on load error", res)
	}
	if mock.calls != 0 {
		t.Fatalf("classifier called %d times, want 0 on load error", mock.calls)
	}
}
