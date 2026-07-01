// Ollama local chat API provider.
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

const defaultOllamaBaseURL = "http://127.0.0.1:11434"

type ollamaProvider struct {
	model   string
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

func newOllamaProvider(cfg config.LLMConfig, apiKey string) *ollamaProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	return &ollamaProvider{
		model:   cfg.Model,
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
		client:  &http.Client{},
	}
}

func (p *ollamaProvider) Classify(ctx context.Context, req ClassifyRequest) (policy.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	body, err := json.Marshal(ollamaRequest{
		Model: p.model,
		Messages: []ollamaMessage{
			{Role: "system", Content: req.Signature},
			{Role: "user", Content: buildUserMessage(req)},
		},
		Stream: false,
	})
	if err != nil {
		return policy.Result{}, fmt.Errorf("marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return policy.Result{}, fmt.Errorf("build ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return policy.Result{}, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return policy.Result{}, fmt.Errorf("read ollama response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return policy.Result{}, fmt.Errorf("ollama API %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed ollamaResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return policy.Result{}, fmt.Errorf("decode ollama response: %w", err)
	}
	if strings.TrimSpace(parsed.Message.Content) == "" {
		return policy.Result{}, fmt.Errorf("empty ollama response")
	}

	return parseClassifyResponse(parsed.Message.Content)
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}
