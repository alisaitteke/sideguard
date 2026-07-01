// Provider factory selects an LLM backend from config.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-2.0-providers.md).
package llm

import (
	"fmt"

	"github.com/alisaitteke/vibeguard/internal/config"
)

// NewProvider constructs a Provider for the configured backend.
func NewProvider(cfg config.LLMConfig, creds config.Credentials) (Provider, error) {
	switch cfg.Provider {
	case "openai":
		return newOpenAIProvider(cfg, creds.OpenAI.APIKey), nil
	case "anthropic":
		return newAnthropicProvider(cfg, creds.Anthropic.APIKey), nil
	case "ollama":
		return newOllamaProvider(cfg, creds.Ollama.APIKey), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %q", cfg.Provider)
	}
}
