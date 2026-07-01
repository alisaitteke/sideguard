package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/alisaitteke/vibeguard/internal/paths"
)

const (
	envOpenAIKey    = "VIBEGUARD_OPENAI_API_KEY"
	envAnthropicKey = "VIBEGUARD_ANTHROPIC_API_KEY"
	envOllamaKey    = "VIBEGUARD_OLLAMA_API_KEY"
)

// Credentials holds provider API keys loaded from disk with env overrides.
type Credentials struct {
	OpenAI    ProviderCredential `yaml:"openai"`
	Anthropic ProviderCredential `yaml:"anthropic"`
	Ollama    ProviderCredential `yaml:"ollama"`
}

// ProviderCredential is a single provider's secret fields.
type ProviderCredential struct {
	APIKey string `yaml:"api_key"`
}

type credentialsFile struct {
	OpenAI    ProviderCredential `yaml:"openai"`
	Anthropic ProviderCredential `yaml:"anthropic"`
	Ollama    ProviderCredential `yaml:"ollama"`
}

// DefaultCredentialsTemplate is written on install when no credentials file exists.
const DefaultCredentialsTemplate = `openai:
  api_key: ""
anthropic:
  api_key: ""
ollama:
  api_key: ""
`

// ResolveCredentials reads ~/.vibeguard/credentials.yaml and applies env overrides.
// Env vars take precedence: VIBEGUARD_OPENAI_API_KEY, VIBEGUARD_ANTHROPIC_API_KEY,
// VIBEGUARD_OLLAMA_API_KEY.
func ResolveCredentials() (Credentials, error) {
	var creds Credentials

	path, err := paths.CredentialsPath()
	if err != nil {
		return creds, err
	}

	if data, err := os.ReadFile(path); err == nil {
		var doc credentialsFile
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return creds, fmt.Errorf("parse credentials %s: %w", path, err)
		}
		creds = Credentials(doc)
	} else if !errors.Is(err, os.ErrNotExist) {
		return creds, fmt.Errorf("read credentials %s: %w", path, err)
	}

	applyEnvOverrides(&creds)
	return creds, nil
}

func applyEnvOverrides(creds *Credentials) {
	if v := os.Getenv(envOpenAIKey); v != "" {
		creds.OpenAI.APIKey = v
	}
	if v := os.Getenv(envAnthropicKey); v != "" {
		creds.Anthropic.APIKey = v
	}
	if v := os.Getenv(envOllamaKey); v != "" {
		creds.Ollama.APIKey = v
	}
}

// EnsureCredentialsDefault writes the default credentials template when missing.
func EnsureCredentialsDefault() (string, error) {
	path, err := paths.CredentialsPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(DefaultCredentialsTemplate), 0o600); err != nil {
		return "", err
	}
	return path, nil
}
