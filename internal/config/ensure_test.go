package config

import "testing"

func TestHasAPIKeyForProvider(t *testing.T) {
	creds := Credentials{
		OpenAI:    ProviderCredential{APIKey: "sk-test"},
		Anthropic: ProviderCredential{APIKey: ""},
	}
	if !HasAPIKeyForProvider("openai", creds) {
		t.Fatal("expected openai key present")
	}
	if HasAPIKeyForProvider("anthropic", creds) {
		t.Fatal("expected anthropic key missing")
	}
	if !HasAPIKeyForProvider("ollama", creds) {
		t.Fatal("ollama should not require API key")
	}
}
