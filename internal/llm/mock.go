// MockProvider returns a fixed classification result for tests and downstream phases.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-2.0-providers.md).
package llm

import (
	"context"

	"github.com/alisaitteke/vibeguard/internal/policy"
)

// MockProvider implements Provider with a predetermined result or error.
type MockProvider struct {
	Result policy.Result
	Err    error
}

// Classify returns the configured result without calling an external API.
func (m *MockProvider) Classify(ctx context.Context, req ClassifyRequest) (policy.Result, error) {
	if m.Err != nil {
		return policy.Result{}, m.Err
	}
	return m.Result, nil
}
