package policy_test

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/alisaitteke/sideguard/internal/approvalmode"
	"github.com/alisaitteke/sideguard/internal/policy"

	_ "github.com/alisaitteke/sideguard/internal/detect"
)

type testClassifier struct {
	calls  int
	result policy.Result
}

func (m *testClassifier) Classify(ctx context.Context, input policy.Input, yamlReason string) policy.Result {
	m.calls++
	return m.result
}

func TestEvaluateFullGitStatusAutoAllow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	res := policy.EvaluateFull(context.Background(), "/tmp", policy.Input{Command: "git status", CWD: "/tmp"}, policy.EvaluateOpts{
		Mode: approvalmode.Auto,
	})
	if res.Action != policy.ActionAllow {
		t.Fatalf("got %+v, want allow for safe git status", res)
	}
	if res.DetectAction != policy.ActionAllow {
		t.Fatalf("detect action = %q, want allow", res.DetectAction)
	}
}

func TestEvaluateFullObfuscatedRmDeny(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	payload := base64.StdEncoding.EncodeToString([]byte("rm -rf /"))
	cmd := "echo " + payload + " | base64 -d | sh"
	res := policy.EvaluateFull(context.Background(), "/tmp", policy.Input{Command: cmd, CWD: "/tmp"}, policy.EvaluateOpts{
		Mode: approvalmode.Auto,
	})
	if res.Action != policy.ActionDeny {
		t.Fatalf("got %+v, want deny for obfuscated rm", res)
	}
	if res.DetectAction != policy.ActionDeny {
		t.Fatalf("detect action = %q, want deny", res.DetectAction)
	}
	if len(res.DetectRules) == 0 {
		t.Fatal("expected matched detect rules")
	}
}

func TestEvaluateFullUnknownAutoLLMOffAsk(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mock := &testClassifier{result: policy.Result{Action: policy.ActionAllow, Reason: "llm should not run"}}
	res := policy.EvaluateFull(context.Background(), "/tmp", policy.Input{Command: "novel-script-xyz", CWD: "/tmp"}, policy.EvaluateOpts{
		Mode:       approvalmode.Auto,
		LLMEnabled: false,
		Classifier: mock,
	})
	if res.Action != policy.ActionAsk {
		t.Fatalf("got %+v, want ask", res)
	}
	if mock.calls != 0 {
		t.Fatalf("classifier called %d times, want 0 when LLM disabled", mock.calls)
	}
}

func TestEvaluateFullYAMLDenySkipsDetect(t *testing.T) {
	home := t.TempDir()
	sideguardDir := filepath.Join(home, ".sideguard")
	if err := os.MkdirAll(sideguardDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `rules:
  - match: { command: "rm -rf" }
    action: deny
    reason: yaml blocked
`
	if err := os.WriteFile(filepath.Join(sideguardDir, "policy.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	res := policy.EvaluateFull(context.Background(), "/tmp", policy.Input{Command: "rm -rf /tmp/foo", CWD: "/tmp"}, policy.EvaluateOpts{
		Mode: approvalmode.Auto,
	})
	if res.Action != policy.ActionDeny || res.Reason != "yaml blocked" {
		t.Fatalf("got %+v, want yaml deny", res)
	}
	if res.DetectAction != "" {
		t.Fatalf("detect should not run on yaml deny, got detect action %q", res.DetectAction)
	}
}

func TestEvaluateFullAutoLLMOnAskInvokesClassifier(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mock := &testClassifier{result: policy.Result{Action: policy.ActionAllow, Reason: "llm approved"}}
	res := policy.EvaluateFull(context.Background(), "/tmp", policy.Input{Command: "novel-script-xyz", CWD: "/tmp"}, policy.EvaluateOpts{
		Mode:       approvalmode.Auto,
		LLMEnabled: true,
		Classifier: mock,
	})
	if res.Action != policy.ActionAllow || res.Reason != "llm approved" {
		t.Fatalf("got %+v, want llm allow", res)
	}
	if mock.calls != 1 {
		t.Fatalf("classifier called %d times, want 1", mock.calls)
	}
}

func TestEvaluateFullAskModeSkipsLLM(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mock := &testClassifier{result: policy.Result{Action: policy.ActionAllow, Reason: "llm should not run"}}
	res := policy.EvaluateFull(context.Background(), "/tmp", policy.Input{Command: "novel-script-xyz", CWD: "/tmp"}, policy.EvaluateOpts{
		Mode:       approvalmode.Ask,
		LLMEnabled: true,
		Classifier: mock,
	})
	if res.Action != policy.ActionAsk {
		t.Fatalf("got %+v, want ask in ask mode", res)
	}
	if mock.calls != 0 {
		t.Fatalf("classifier called %d times, want 0 in ask mode", mock.calls)
	}
}

func TestEvaluateFullAutoDistinctFromAutoAllow(t *testing.T) {
	if approvalmode.Auto == approvalmode.AutoAllow {
		t.Fatal("auto and auto_allow must be distinct modes")
	}
	if approvalmode.Auto.Decision() != "" {
		t.Fatalf("auto Decision = %q, want empty (hook uses detect)", approvalmode.Auto.Decision())
	}
	if approvalmode.AutoAllow.Decision() != "allow" {
		t.Fatalf("auto_allow Decision = %q, want allow", approvalmode.AutoAllow.Decision())
	}
}
