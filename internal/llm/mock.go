// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// MockProvider and MockChatDriver return fixed results for tests.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-2.0-llm.md).
package llm

import (
	"context"

	"github.com/alisaitteke/sideguard/internal/policy"
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

// MockChatDriver implements ChatDriver with a fixed response or error.
type MockChatDriver struct {
	Content string
	Err     error
}

// Chat returns the configured content without calling an external API.
func (m *MockChatDriver) Chat(ctx context.Context, req ChatRequest) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	return m.Content, nil
}

// MockAnalyzer implements Analyzer with a fixed result or error.
type MockAnalyzer struct {
	Result AnalyzeResult
	Err    error
}

// Analyze returns the configured result without calling an external API.
func (m *MockAnalyzer) Analyze(ctx context.Context, input AnalyzeInput) (AnalyzeResult, error) {
	if m.Err != nil {
		return AnalyzeResult{}, m.Err
	}
	return m.Result, nil
}
