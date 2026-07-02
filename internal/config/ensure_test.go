package config

import "testing"

func TestHasAPIKeyForProvider(t *testing.T) {
	creds := map[string]ProviderCredential{
		"my-openai":    {APIKey: "sk-test"},
		"my-anthropic": {APIKey: ""},
	}
	if !HasAPIKeyForProvider("openai", creds, "my-openai") {
		t.Fatal("expected openai instance key present")
	}
	if HasAPIKeyForProvider("anthropic", creds, "my-anthropic") {
		t.Fatal("expected anthropic instance key missing")
	}
	if !HasAPIKeyForProvider("ollama", creds, "my-ollama") {
		t.Fatal("ollama should not require API key")
	}
}
