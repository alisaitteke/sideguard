// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Chat primitive for provider-driven LLM backends.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-2.0-llm.md).
package llm

import "context"

// ChatRequest is the minimal input for a single LLM chat turn.
type ChatRequest struct {
	SystemPrompt string
	UserPrompt   string
}

// ChatDriver performs one system+user chat completion against a backend.
type ChatDriver interface {
	Chat(ctx context.Context, req ChatRequest) (string, error)
}
