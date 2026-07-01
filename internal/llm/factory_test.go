package llm

import (
	"testing"

	"github.com/alisaitteke/vibeguard/internal/config"
)

func TestNewProviderKnown(t *testing.T) {
	t.Parallel()

	creds := config.Credentials{
		OpenAI:    config.ProviderCredential{APIKey: "sk-test"},
		Anthropic: config.ProviderCredential{APIKey: "ant-test"},
	}

	cases := []struct {
		name     string
		provider string
	}{
		{"openai", "openai"},
		{"anthropic", "anthropic"},
		{"ollama", "ollama"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.LLMConfig{
				Provider:  tc.provider,
				Model:     "test-model",
				TimeoutMS: 3000,
			}
			p, err := NewProvider(cfg, creds)
			if err != nil {
				t.Fatalf("NewProvider: %v", err)
			}
			if p == nil {
				t.Fatal("expected non-nil provider")
			}
		})
	}
}

func TestNewProviderUnknown(t *testing.T) {
	t.Parallel()

	_, err := NewProvider(config.LLMConfig{Provider: "unknown-vendor"}, config.Credentials{})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
