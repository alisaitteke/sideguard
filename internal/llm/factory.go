// Provider factory resolves multi-provider LLM settings to a Classify backend.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-2.0-llm.md).
package llm

import (
	"context"
	"fmt"

	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

const defaultClassifySignature = "default"

// NewProvider constructs a Provider for the default (or explicit) provider instance.
func NewProvider(settings config.LLMSettings, creds map[string]config.ProviderCredential) (Provider, error) {
	return NewProviderFor(settings, creds, settings.DefaultProvider)
}

// NewProviderFor constructs a Provider for a specific provider instance id.
func NewProviderFor(settings config.LLMSettings, creds map[string]config.ProviderCredential, providerID string) (Provider, error) {
	instance, err := resolveProviderInstance(settings, providerID)
	if err != nil {
		return nil, err
	}

	cred, ok := creds[instance.ID]
	if !ok {
		cred = config.ProviderCredential{}
	}

	driver, err := NewChatDriver(instance, cred, settings.TimeoutMS)
	if err != nil {
		return nil, fmt.Errorf("provider %q: %w", instance.ID, err)
	}

	return &chatProvider{driver: driver}, nil
}

type chatProvider struct {
	driver ChatDriver
}

func (p *chatProvider) Classify(ctx context.Context, req ClassifyRequest) (policy.Result, error) {
	content, err := p.driver.Chat(ctx, ChatRequest{
		SystemPrompt: req.Signature,
		UserPrompt:   buildUserMessage(req),
	})
	if err != nil {
		return policy.Result{}, err
	}
	return parseClassifyResponse(content)
}
