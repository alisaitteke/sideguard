// OpenAI-compatible chat completions provider (OpenAI, Azure, OpenRouter, etc.).
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

const defaultOpenAIBaseURL = "https://api.openai.com"

type openAIProvider struct {
	model   string
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

func newOpenAIProvider(cfg config.LLMConfig, apiKey string) *openAIProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &openAIProvider{
		model:   cfg.Model,
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
		client:  &http.Client{},
	}
}

func (p *openAIProvider) Classify(ctx context.Context, req ClassifyRequest) (policy.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	body, err := json.Marshal(openAIRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{Role: "system", Content: req.Signature},
			{Role: "user", Content: buildUserMessage(req)},
		},
	})
	if err != nil {
		return policy.Result{}, fmt.Errorf("marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return policy.Result{}, fmt.Errorf("build openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return policy.Result{}, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return policy.Result{}, fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return policy.Result{}, fmt.Errorf("openai API %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed openAIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return policy.Result{}, fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return policy.Result{}, fmt.Errorf("empty openai response")
	}

	return parseClassifyResponse(parsed.Choices[0].Message.Content)
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}
