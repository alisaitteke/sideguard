package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alisaitteke/sideguard/internal/config"
	"github.com/alisaitteke/sideguard/internal/policy"
)

func TestAnthropicProviderClassify(t *testing.T) {
	t.Parallel()

	const modelContent = `{"action":"deny","reason":"destructive rm"}`

	var gotKey, gotVer string
	var gotBody anthropicRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path = %q, want /v1/messages", r.URL.Path)
		}
		gotKey = r.Header.Get("x-api-key")
		gotVer = r.Header.Get("anthropic-version")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		_ = json.NewEncoder(w).Encode(anthropicResponse{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: modelContent}},
		})
	}))
	defer srv.Close()

	driver, err := newAnthropicChatDriver(driverConfig{
		instance: config.ProviderInstance{Model: "claude-3-5-haiku-latest", BaseURL: srv.URL},
		apiKey:   "ant-test-key",
		timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("newAnthropicChatDriver: %v", err)
	}
	p := &chatProvider{driver: driver}

	req := ClassifyRequest{
		Input:      policy.Input{Command: "rm -rf /"},
		YAMLAction: policy.ActionAsk,
		Signature:  "classify as JSON",
	}

	result, err := p.Classify(context.Background(), req)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if result.Action != policy.ActionDeny {
		t.Errorf("action = %q, want deny", result.Action)
	}
	if result.Reason != "destructive rm" {
		t.Errorf("reason = %q", result.Reason)
	}

	if gotKey != "ant-test-key" {
		t.Errorf("x-api-key = %q", gotKey)
	}
	if gotVer != anthropicVersion {
		t.Errorf("anthropic-version = %q", gotVer)
	}
	if gotBody.System != req.Signature {
		t.Errorf("system = %q", gotBody.System)
	}
	if len(gotBody.Messages) != 1 || gotBody.Messages[0].Role != "user" {
		t.Errorf("messages = %+v", gotBody.Messages)
	}
	if !strings.Contains(gotBody.Messages[0].Content, "rm -rf") {
		t.Errorf("user content = %s", gotBody.Messages[0].Content)
	}
}

func TestAnthropicProviderHTTP403(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	driver, err := newAnthropicChatDriver(driverConfig{
		instance: config.ProviderInstance{Model: "m", BaseURL: srv.URL},
		apiKey:   "bad-key",
		timeout:  3 * time.Second,
	})
	if err != nil {
		t.Fatalf("newAnthropicChatDriver: %v", err)
	}
	p := &chatProvider{driver: driver}

	_, err = p.Classify(context.Background(), ClassifyRequest{
		Signature:  "sys",
		YAMLAction: policy.ActionAsk,
	})
	if err == nil {
		t.Fatal("expected error on 403")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %v", err)
	}
}
