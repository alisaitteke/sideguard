// Anthropic Messages API provider.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-2.0-llm.md).
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
)

const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	anthropicVersion        = "2023-06-01"
	anthropicMaxTokens      = 256
)

type anthropicChatDriver struct {
	model   string
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

func newAnthropicChatDriver(cfg driverConfig) (ChatDriver, error) {
	baseURL := cfg.instance.BaseURL
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	return &anthropicChatDriver{
		model:   cfg.instance.Model,
		apiKey:  cfg.apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		timeout: cfg.timeout,
		client:  &http.Client{},
	}, nil
}

func (d *anthropicChatDriver) Chat(ctx context.Context, req ChatRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	body, err := json.Marshal(anthropicRequest{
		Model:     d.model,
		MaxTokens: anthropicMaxTokens,
		System:    req.SystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: req.UserPrompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", d.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic API %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode anthropic response: %w", err)
	}
	if len(parsed.Content) == 0 || strings.TrimSpace(parsed.Content[0].Text) == "" {
		return "", fmt.Errorf("empty anthropic response")
	}

	return parsed.Content[0].Text, nil
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
