// Package llm defines contracts for provider-driven LLM command classification.
// HTTP providers are implemented in later phases; this package holds types and loaders.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-1.0-contracts.md).
package llm

import "github.com/alisaitteke/vibeguard/internal/policy"

// ClassifyRequest is the input passed to an LLM provider for triage.
type ClassifyRequest struct {
	Input      policy.Input
	YAMLAction policy.Action // always ActionAsk when LLM is invoked
	YAMLReason string
	Signature  string // resolved system prompt body
}
