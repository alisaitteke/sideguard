// EvaluateWithLLM combines deterministic YAML policy with optional LLM triage.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-4.0-integration.md).
package policy

import "context"

// Classifier runs LLM classification after YAML returns ask or no rule match.
type Classifier interface {
	Classify(ctx context.Context, input Input, yamlReason string) Result
}

// EvaluateWithLLM runs YAML Evaluate first, then invokes clf when YAML returns ask
// (including no-match default) and llmEnabled is true. YAML allow/deny never call LLM.
func EvaluateWithLLM(ctx context.Context, cwd string, input Input, clf Classifier, llmEnabled bool) Result {
	yaml := Evaluate(cwd, input)

	if yaml.Action == ActionAllow || yaml.Action == ActionDeny {
		return yaml
	}

	if !llmEnabled {
		return yaml
	}

	if clf == nil {
		return yaml
	}

	return clf.Classify(ctx, input, yaml.Reason)
}
