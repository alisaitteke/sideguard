// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package llm

import (
	"context"

	"github.com/alisaitteke/sideguard/internal/policy"
)

// Provider classifies shell commands and MCP tool calls via an LLM backend.
// Concrete implementations (OpenAI, Anthropic, Ollama) ship in lat-phase-2.0-providers.
type Provider interface {
	Classify(ctx context.Context, req ClassifyRequest) (policy.Result, error)
}
