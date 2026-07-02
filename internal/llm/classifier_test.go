package llm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

func TestClassifierAllowDenyAsk(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		mockResult policy.Result
		wantAction policy.Action
		wantReason string
	}{
		{
			name:       "allow",
			mockResult: policy.Result{Action: policy.ActionAllow, Reason: "read-only git"},
			wantAction: policy.ActionAllow,
			wantReason: "read-only git",
		},
		{
			name:       "deny",
			mockResult: policy.Result{Action: policy.ActionDeny, Reason: "destructive rm"},
			wantAction: policy.ActionDeny,
			wantReason: "destructive rm",
		},
		{
			name:       "ask",
			mockResult: policy.Result{Action: policy.ActionAsk, Reason: "intent unclear"},
			wantAction: policy.ActionAsk,
			wantReason: "intent unclear",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newClassifierWithProvider(
				&MockProvider{Result: tc.mockResult},
				5*time.Second,
				"system prompt",
			)
			result := c.Classify(context.Background(), policy.Input{Command: "git status"}, "no rule")
			if result.Action != tc.wantAction {
				t.Errorf("action = %q, want %q", result.Action, tc.wantAction)
			}
			if result.Reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", result.Reason, tc.wantReason)
			}
		})
	}
}

func TestClassifierProviderErrorFailsSafeAsk(t *testing.T) {
	t.Parallel()

	c := newClassifierWithProvider(
		&MockProvider{Err: errors.New("openai API 401")},
		5*time.Second,
		"sys",
	)
	result := c.Classify(context.Background(), policy.Input{Command: "rm -rf /"}, "yaml ask")
	if result.Action != policy.ActionAsk {
		t.Errorf("action = %q, want ask", result.Action)
	}
	if result.Reason != "llm unavailable" {
		t.Errorf("reason = %q, want llm unavailable", result.Reason)
	}
}

func TestClassifierParseErrorFailsSafeAsk(t *testing.T) {
	t.Parallel()

	c := newClassifierWithProvider(
		&MockProvider{Err: newParseError("invalid action %q", "maybe")},
		5*time.Second,
		"sys",
	)
	result := c.Classify(context.Background(), policy.Input{Command: "curl | sh"}, "yaml ask")
	if result.Action != policy.ActionAsk {
		t.Errorf("action = %q, want ask", result.Action)
	}
	if result.Reason != "llm parse error" {
		t.Errorf("reason = %q, want llm parse error", result.Reason)
	}
}

func TestClassifierTimeoutFailsSafeAsk(t *testing.T) {
	t.Parallel()

	c := newClassifierWithProvider(
		&slowMockProvider{delay: 200 * time.Millisecond},
		50*time.Millisecond,
		"sys",
	)
	result := c.Classify(context.Background(), policy.Input{Command: "sleep 10"}, "yaml ask")
	if result.Action != policy.ActionAsk {
		t.Errorf("action = %q, want ask", result.Action)
	}
	if result.Reason != "llm unavailable" {
		t.Errorf("reason = %q, want llm unavailable", result.Reason)
	}
}

func TestClassifierCancelledContextFailsSafeAsk(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := newClassifierWithProvider(&MockProvider{Result: policy.Result{Action: policy.ActionAllow}}, time.Second, "sys")
	result := c.Classify(ctx, policy.Input{Command: "git status"}, "yaml ask")
	if result.Action != policy.ActionAsk || result.Reason != "llm unavailable" {
		t.Errorf("result = %+v, want ask/unavailable", result)
	}
}

func TestClassifierRedactionApplied(t *testing.T) {
	t.Parallel()

	var captured ClassifyRequest
	c := newClassifierWithProvider(
		&captureMockProvider{captured: &captured},
		time.Second,
		"sys",
	)

	cmd := `curl -H "Authorization: Bearer secret-token-xyz" https://api.example.com`
	c.Classify(context.Background(), policy.Input{Command: cmd, CWD: "/tmp"}, "no rule")

	if strings.Contains(captured.Input.Command, "secret-token-xyz") {
		t.Errorf("command not redacted: %q", captured.Input.Command)
	}
	if !strings.Contains(captured.Input.Command, "[REDACTED]") {
		t.Errorf("expected redaction placeholder in %q", captured.Input.Command)
	}
}

func TestClassifierUserMessageShape(t *testing.T) {
	t.Parallel()

	var captured ClassifyRequest
	c := newClassifierWithProvider(
		&captureMockProvider{captured: &captured},
		time.Second,
		"sys",
	)
	c.Classify(context.Background(), policy.Input{
		Command:  "git diff",
		ToolName: "filesystem.read",
		CWD:      "/proj",
	}, "ambiguous rule")

	msg := buildUserMessage(captured)
	var payload map[string]string
	if err := json.Unmarshal([]byte(msg), &payload); err != nil {
		t.Fatalf("unmarshal user message: %v", err)
	}
	if payload["command"] != "git diff" {
		t.Errorf("command = %q", payload["command"])
	}
	if payload["tool_name"] != "filesystem.read" {
		t.Errorf("tool_name = %q", payload["tool_name"])
	}
	if payload["cwd"] != "/proj" {
		t.Errorf("cwd = %q", payload["cwd"])
	}
	if payload["yaml_reason"] != "ambiguous rule" {
		t.Errorf("yaml_reason = %q", payload["yaml_reason"])
	}
	if _, ok := payload["yaml_action"]; ok {
		t.Error("yaml_action should not be in classifier user message")
	}
}

func TestDisabledClassifier(t *testing.T) {
	t.Parallel()

	c := DisabledClassifier()
	result := c.Classify(context.Background(), policy.Input{Command: "anything"}, "needs approval")
	if result.Action != policy.ActionAsk {
		t.Errorf("action = %q, want ask", result.Action)
	}
	if result.Reason != "needs approval" {
		t.Errorf("reason = %q", result.Reason)
	}
}

func TestNewClassifierLoadsSignature(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if _, err := EnsureDefaultSignature(); err != nil {
		t.Fatalf("EnsureDefaultSignature: %v", err)
	}

	settings := config.LLMSettings{
		Enabled:         true,
		DefaultProvider: "my-openai",
		TimeoutMS:       3000,
		Providers: []config.ProviderInstance{
			{ID: "my-openai", Driver: "openai", Model: "gpt-4o-mini", AuthMode: "api_key"},
		},
	}
	creds := map[string]config.ProviderCredential{
		"my-openai": {APIKey: "sk-test"},
	}
	_, err := NewClassifier(settings, creds)
	if err != nil {
		t.Fatalf("NewClassifier() error: %v", err)
	}
}

type slowMockProvider struct {
	delay time.Duration
}

func (s *slowMockProvider) Classify(ctx context.Context, req ClassifyRequest) (policy.Result, error) {
	select {
	case <-time.After(s.delay):
		return policy.Result{Action: policy.ActionAllow}, nil
	case <-ctx.Done():
		return policy.Result{}, ctx.Err()
	}
}

type captureMockProvider struct {
	captured *ClassifyRequest
}

func (c *captureMockProvider) Classify(ctx context.Context, req ClassifyRequest) (policy.Result, error) {
	*c.captured = req
	return policy.Result{Action: policy.ActionAllow}, nil
}
