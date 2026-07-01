package llm

import (
	"testing"

	"github.com/alisaitteke/vibeguard/internal/policy"
)

func TestMockProvider(t *testing.T) {
	t.Parallel()

	p := &MockProvider{
		Result: policy.Result{Action: policy.ActionAllow, Reason: "mocked"},
	}
	result, err := p.Classify(t.Context(), ClassifyRequest{})
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if result.Action != policy.ActionAllow || result.Reason != "mocked" {
		t.Errorf("result = %+v", result)
	}
}
