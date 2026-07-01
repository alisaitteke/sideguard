package config

// HasAPIKeyForProvider reports whether credentials are configured for provider.
// Ollama local mode does not require an API key.
func HasAPIKeyForProvider(provider string, creds Credentials) bool {
	switch provider {
	case "ollama":
		return true
	case "anthropic":
		return creds.Anthropic.APIKey != ""
	case "openai":
		return creds.OpenAI.APIKey != ""
	default:
		return false
	}
}
