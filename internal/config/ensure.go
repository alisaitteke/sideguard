package config

// HasAPIKeyForProvider reports whether credentials are configured for a provider instance.
// Ollama local mode does not require an API key.
func HasAPIKeyForProvider(driver string, creds map[string]ProviderCredential, providerID string) bool {
	if driver == "ollama" {
		return true
	}
	c, ok := creds[providerID]
	return ok && c.APIKey != ""
}
