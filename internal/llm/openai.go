// OpenAI-compatible chat completions provider (OpenAI, Azure, OpenRouter, etc.).
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

const defaultOpenAIBaseURL = "https://api.openai.com"

type openAIChatDriver struct {
	model   string
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

func newOpenAIChatDriver(cfg driverConfig) (ChatDriver, error) {
	baseURL := cfg.instance.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &openAIChatDriver{
		model:   cfg.instance.Model,
		apiKey:  cfg.apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		timeout: cfg.timeout,
		client:  &http.Client{},
	}, nil
}

func (d *openAIChatDriver) Chat(ctx context.Context, req ChatRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	body, err := json.Marshal(openAIRequest{
		Model: d.model,
		Messages: []openAIMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserPrompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if d.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)
	}

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai API %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed openAIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("empty openai response")
	}

	return parsed.Choices[0].Message.Content, nil
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
