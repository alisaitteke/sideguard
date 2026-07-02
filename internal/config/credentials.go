package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/alisaitteke/sideguard/internal/paths"
)

const (
	envOpenAIKey    = "SIDEGUARD_OPENAI_API_KEY"
	envAnthropicKey = "SIDEGUARD_ANTHROPIC_API_KEY"
	envOllamaKey    = "SIDEGUARD_OLLAMA_API_KEY"
)

// ProviderCredential is a single provider instance secret fields.
type ProviderCredential struct {
	APIKey string `yaml:"api_key"`
}

type credentialsFile struct {
	Providers map[string]ProviderCredential `yaml:"providers"`
}

// DefaultCredentialsTemplate is written on install when no credentials file exists.
const DefaultCredentialsTemplate = `providers: {}
`

// ResolveProviderCredentials reads ~/.sideguard/credentials.yaml and applies per-driver env overrides.
// Env vars (SIDEGUARD_OPENAI_API_KEY, SIDEGUARD_ANTHROPIC_API_KEY, SIDEGUARD_OLLAMA_API_KEY) apply to
// every configured provider instance with a matching driver when the env value is set.
func ResolveProviderCredentials() (map[string]ProviderCredential, error) {
	creds := make(map[string]ProviderCredential)

	path, err := paths.CredentialsPath()
	if err != nil {
		return creds, err
	}

	if data, err := os.ReadFile(path); err == nil {
		var doc credentialsFile
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return creds, fmt.Errorf("parse credentials %s: %w", path, err)
		}
		if doc.Providers != nil {
			for id, c := range doc.Providers {
				creds[id] = c
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return creds, fmt.Errorf("read credentials %s: %w", path, err)
	}

	if err := applyEnvOverrides(creds); err != nil {
		return creds, err
	}
	return creds, nil
}

func applyEnvOverrides(creds map[string]ProviderCredential) error {
	settings, err := LoadLLMSettings("")
	if err != nil {
		return err
	}
	for _, p := range settings.Providers {
		envKey := envKeyForDriver(p.Driver)
		if envKey == "" {
			continue
		}
		if v := os.Getenv(envKey); v != "" {
			c := creds[p.ID]
			c.APIKey = v
			creds[p.ID] = c
		}
	}
	return nil
}

func envKeyForDriver(driver string) string {
	switch driver {
	case "openai", "openai-compatible":
		return envOpenAIKey
	case "anthropic":
		return envAnthropicKey
	case "ollama":
		return envOllamaKey
	default:
		return ""
	}
}

// SetProviderKey writes or updates an instance API key in credentials.yaml (mode 0600).
func SetProviderKey(id, apiKey string) error {
	if id == "" {
		return errors.New("provider id required")
	}

	path, err := paths.CredentialsPath()
	if err != nil {
		return err
	}

	doc := credentialsFile{Providers: make(map[string]ProviderCredential)}
	if data, readErr := os.ReadFile(path); readErr == nil {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse credentials %s: %w", path, err)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("read credentials %s: %w", path, readErr)
	}
	if doc.Providers == nil {
		doc.Providers = make(map[string]ProviderCredential)
	}

	doc.Providers[id] = ProviderCredential{APIKey: apiKey}
	return writeCredentialsFile(path, doc)
}

func removeProviderCredential(id string) error {
	path, err := paths.CredentialsPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read credentials %s: %w", path, err)
	}

	var doc credentialsFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse credentials %s: %w", path, err)
	}
	if doc.Providers == nil {
		return nil
	}
	delete(doc.Providers, id)
	return writeCredentialsFile(path, doc)
}

func writeCredentialsFile(path string, doc credentialsFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("write credentials %s: %w", path, err)
	}
	return nil
}

// ProviderStatus returns a masked API key and whether credentials are configured for id.
func ProviderStatus(id string) (maskedKey string, configured bool, err error) {
	creds, err := ResolveProviderCredentials()
	if err != nil {
		return "", false, err
	}
	c, ok := creds[id]
	if !ok || c.APIKey == "" {
		return "", false, nil
	}
	return maskAPIKey(c.APIKey), true, nil
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "…" + key[len(key)-min(4, len(key)):]
	}
	if strings.HasPrefix(key, "sk-") && len(key) > 11 {
		return key[:7] + "…" + key[len(key)-4:]
	}
	return key[:3] + "…" + key[len(key)-4:]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
