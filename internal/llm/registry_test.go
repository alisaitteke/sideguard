package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

func TestRegisteredDrivers(t *testing.T) {
	t.Parallel()

	drivers := RegisteredDrivers()
	if len(drivers) < 4 {
		t.Fatalf("expected at least 4 drivers, got %d", len(drivers))
	}

	names := make(map[string]DriverInfo)
	for _, d := range drivers {
		names[d.Name] = d
	}

	for _, want := range []string{"openai", "anthropic", "ollama", "openai-compatible"} {
		info, ok := names[want]
		if !ok {
			t.Fatalf("missing driver %q", want)
		}
		if info.Label == "" {
			t.Errorf("driver %q missing label", want)
		}
	}
}

func TestNewChatDriverUnknown(t *testing.T) {
	t.Parallel()

	_, err := NewChatDriver(config.ProviderInstance{
		ID: "x", Driver: "nonexistent", Model: "m", AuthMode: "api_key",
	}, config.ProviderCredential{APIKey: "k"}, 3000)
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
}

func TestValidateAuthMode(t *testing.T) {
	t.Parallel()

	if err := ValidateAuthMode("api_key"); err != nil {
		t.Fatalf("api_key: %v", err)
	}
	if err := ValidateAuthMode("subscription"); !errors.Is(err, ErrAuthNotImplemented) {
		t.Fatalf("subscription: %v", err)
	}
	if err := ValidateAuthMode("bogus"); err == nil {
		t.Fatal("expected error for bogus auth_mode")
	}
}

func TestChatProviderClassifyRegression(t *testing.T) {
	t.Parallel()

	p := &chatProvider{driver: &MockChatDriver{
		Content: `{"action":"allow","reason":"read-only git"}`,
	}}
	result, err := p.Classify(context.Background(), ClassifyRequest{
		Input:      policy.Input{Command: "git status"},
		YAMLAction: policy.ActionAsk,
		Signature:  "sys",
	})
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if result.Action != policy.ActionAllow {
		t.Errorf("action = %q, want allow", result.Action)
	}
}
