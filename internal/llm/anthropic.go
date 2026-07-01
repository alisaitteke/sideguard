// Anthropic Messages API provider.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-2.0-providers.md).
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	anthropicVersion        = "2023-06-01"
	anthropicMaxTokens      = 256
)

type anthropicProvider struct {
	model   string
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

func newAnthropicProvider(cfg config.LLMConfig, apiKey string) *anthropicProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	return &anthropicProvider{
		model:   cfg.Model,
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
		client:  &http.Client{},
	}
}

func (p *anthropicProvider) Classify(ctx context.Context, req ClassifyRequest) (policy.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	body, err := json.Marshal(anthropicRequest{
		Model:     p.model,
		MaxTokens: anthropicMaxTokens,
		System:    req.Signature,
		Messages: []anthropicMessage{
			{Role: "user", Content: buildUserMessage(req)},
		},
	})
	if err != nil {
		return policy.Result{}, fmt.Errorf("marshal anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return policy.Result{}, fmt.Errorf("build anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return policy.Result{}, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return policy.Result{}, fmt.Errorf("read anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return policy.Result{}, fmt.Errorf("anthropic API %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return policy.Result{}, fmt.Errorf("decode anthropic response: %w", err)
	}
	if len(parsed.Content) == 0 || strings.TrimSpace(parsed.Content[0].Text) == "" {
		return policy.Result{}, fmt.Errorf("empty anthropic response")
	}

	return parseClassifyResponse(parsed.Content[0].Text)
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}
