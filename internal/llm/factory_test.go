package llm

import (
	"testing"

	"github.com/alisaitteke/sideguard/internal/config"
)

func testLLMSettings(providerID, driver string) config.LLMSettings {
	return config.LLMSettings{
		Enabled:         true,
		DefaultProvider: providerID,
		TimeoutMS:       3000,
		Providers: []config.ProviderInstance{
			{ID: providerID, Driver: driver, Model: "test-model", AuthMode: "api_key"},
		},
	}
}

func TestNewProviderKnown(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		driver   string
		provider string
	}{
		{"openai", "openai", "my-openai"},
		{"anthropic", "anthropic", "my-anthropic"},
		{"ollama", "ollama", "my-ollama"},
		{"openai-compatible", "openai-compatible", "my-compat"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			settings := testLLMSettings(tc.provider, tc.driver)
			creds := map[string]config.ProviderCredential{
				tc.provider: {APIKey: "sk-test"},
			}
			if tc.driver == "ollama" {
				creds[tc.provider] = config.ProviderCredential{}
			}
			p, err := NewProvider(settings, creds)
			if err != nil {
				t.Fatalf("NewProvider: %v", err)
			}
			if p == nil {
				t.Fatal("expected non-nil provider")
			}
		})
	}
}

func TestNewProviderUnknownDriver(t *testing.T) {
	t.Parallel()

	settings := config.LLMSettings{
		DefaultProvider: "bad",
		TimeoutMS:       3000,
		Providers: []config.ProviderInstance{
			{ID: "bad", Driver: "unknown-vendor", Model: "m", AuthMode: "api_key"},
		},
	}
	creds := map[string]config.ProviderCredential{"bad": {APIKey: "key"}}

	_, err := NewProvider(settings, creds)
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
}

func TestNewProviderMissingDefault(t *testing.T) {
	t.Parallel()

	_, err := NewProvider(config.LLMSettings{TimeoutMS: 3000}, nil)
	if err == nil {
		t.Fatal("expected error when default_provider missing")
	}
}

func TestNewProviderSubscriptionAuthMode(t *testing.T) {
	t.Parallel()

	settings := testLLMSettings("sub", "openai")
	settings.Providers[0].AuthMode = "subscription"
	creds := map[string]config.ProviderCredential{"sub": {APIKey: "sk-test"}}

	_, err := NewProvider(settings, creds)
	if err == nil {
		t.Fatal("expected error for subscription auth_mode")
	}
}
