// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Ollama local chat API provider.
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

const defaultOllamaBaseURL = "http://127.0.0.1:11434"

type ollamaChatDriver struct {
	model   string
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

func newOllamaChatDriver(cfg driverConfig) (ChatDriver, error) {
	baseURL := cfg.instance.BaseURL
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	return &ollamaChatDriver{
		model:   cfg.instance.Model,
		apiKey:  cfg.apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		timeout: cfg.timeout,
		client:  &http.Client{},
	}, nil
}

func (d *ollamaChatDriver) Chat(ctx context.Context, req ChatRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	body, err := json.Marshal(ollamaRequest{
		Model: d.model,
		Messages: []ollamaMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserPrompt},
		},
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if d.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)
	}

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read ollama response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed ollamaResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}
	if strings.TrimSpace(parsed.Message.Content) == "" {
		return "", fmt.Errorf("empty ollama response")
	}

	return parsed.Message.Content, nil
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
