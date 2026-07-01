package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

func TestOpenAIProviderClassify(t *testing.T) {
	t.Parallel()

	const modelContent = `{"action":"allow","reason":"read-only git status"}`

	var gotAuth string
	var gotBody openAIRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		_ = json.NewEncoder(w).Encode(openAIResponse{
			Choices: []struct {
				Message openAIMessage `json:"message"`
			}{{Message: openAIMessage{Content: modelContent}}},
		})
	}))
	defer srv.Close()

	cfg := config.LLMConfig{
		Provider:  "openai",
		Model:     "gpt-4o-mini",
		TimeoutMS: 5000,
		BaseURL:   srv.URL,
	}
	p := newOpenAIProvider(cfg, "sk-test-key")

	req := ClassifyRequest{
		Input: policy.Input{
			Command: "git status",
			CWD:     "/tmp/proj",
		},
		YAMLAction: policy.ActionAsk,
		YAMLReason: "no matching rule",
		Signature:  "classify commands as JSON",
	}

	result, err := p.Classify(context.Background(), req)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if result.Action != policy.ActionAllow {
		t.Errorf("action = %q, want allow", result.Action)
	}
	if result.Reason != "read-only git status" {
		t.Errorf("reason = %q", result.Reason)
	}

	if gotAuth != "Bearer sk-test-key" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotBody.Model != "gpt-4o-mini" {
		t.Errorf("model = %q", gotBody.Model)
	}
	if len(gotBody.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(gotBody.Messages))
	}
	if gotBody.Messages[0].Role != "system" || gotBody.Messages[0].Content != req.Signature {
		t.Errorf("system message = %+v", gotBody.Messages[0])
	}
	if gotBody.Messages[1].Role != "user" {
		t.Errorf("user role = %q", gotBody.Messages[1].Role)
	}
	if !strings.Contains(gotBody.Messages[1].Content, "git status") {
		t.Errorf("user content missing command: %s", gotBody.Messages[1].Content)
	}
}

func TestOpenAIProviderHTTP401(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer srv.Close()

	cfg := config.LLMConfig{
		Model:     "gpt-4o-mini",
		TimeoutMS: 3000,
		BaseURL:   srv.URL,
	}
	p := newOpenAIProvider(cfg, "bad-key")

	_, err := p.Classify(context.Background(), ClassifyRequest{
		Signature: "sys",
		YAMLAction: policy.ActionAsk,
	})
	if err == nil {
		t.Fatal("expected error on 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %v", err)
	}
}

func TestOpenAIProviderEmptyResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(openAIResponse{Choices: []struct {
			Message openAIMessage `json:"message"`
		}{{Message: openAIMessage{Content: ""}}}})
	}))
	defer srv.Close()

	cfg := config.LLMConfig{Model: "m", TimeoutMS: 3000, BaseURL: srv.URL}
	p := newOpenAIProvider(cfg, "key")

	_, err := p.Classify(context.Background(), ClassifyRequest{Signature: "sys", YAMLAction: policy.ActionAsk})
	if err == nil {
		t.Fatal("expected error on empty response")
	}
}
