// Classifier orchestrates LLM triage: redaction, provider call, parse, and fail-safe defaults.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-3.0-classifier.md).
package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/alisaitteke/sideguard/internal/config"
	"github.com/alisaitteke/sideguard/internal/policy"
)

// Classifier runs LLM classification after YAML policy returns ask or no match.
type Classifier interface {
	Classify(ctx context.Context, input policy.Input, yamlReason string) policy.Result
}

type classifier struct {
	provider  Provider
	timeout   time.Duration
	signature string
}

// NewClassifier loads the triage signature, constructs a provider, and returns a Classifier.
func NewClassifier(settings config.LLMSettings, creds map[string]config.ProviderCredential) (Classifier, error) {
	sig, err := LoadSignature(defaultClassifySignature)
	if err != nil {
		return nil, fmt.Errorf("load signature %q: %w", defaultClassifySignature, err)
	}

	provider, err := NewProvider(settings, creds)
	if err != nil {
		return nil, err
	}

	return &classifier{
		provider:  provider,
		timeout:   time.Duration(settings.TimeoutMS) * time.Millisecond,
		signature: sig,
	}, nil
}

type disabledClassifier struct{}

// DisabledClassifier returns ask for every input when LLM is off (caller uses YAML only).
func DisabledClassifier() Classifier {
	return disabledClassifier{}
}

func (disabledClassifier) Classify(ctx context.Context, input policy.Input, yamlReason string) policy.Result {
	return policy.Result{Action: policy.ActionAsk, Reason: yamlReason}
}

func (c *classifier) Classify(ctx context.Context, input policy.Input, yamlReason string) policy.Result {
	if err := ctx.Err(); err != nil {
		return failAskUnavailable()
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := ClassifyRequest{
		Input: policy.Input{
			Command:  RedactCommand(input.Command),
			ToolName: input.ToolName,
			CWD:      input.CWD,
		},
		YAMLAction: policy.ActionAsk,
		YAMLReason: yamlReason,
		Signature:  c.signature,
	}

	result, err := c.provider.Classify(ctx, req)
	if err != nil {
		if isParseError(err) {
			return failAskParse()
		}
		return failAskUnavailable()
	}
	return result
}

func failAskParse() policy.Result {
	return policy.Result{Action: policy.ActionAsk, Reason: "llm parse error"}
}

func failAskUnavailable() policy.Result {
	return policy.Result{Action: policy.ActionAsk, Reason: "llm unavailable"}
}

// newClassifierWithProvider constructs a Classifier for unit tests with an injected provider.
func newClassifierWithProvider(provider Provider, timeout time.Duration, signature string) Classifier {
	return &classifier{
		provider:  provider,
		timeout:   timeout,
		signature: signature,
	}
}
