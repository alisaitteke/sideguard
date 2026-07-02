package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alisaitteke/sideguard/internal/config"
	"github.com/alisaitteke/sideguard/internal/llm"
	"github.com/alisaitteke/sideguard/internal/store"
)

func TestAnalyzeCommandMissingFields(t *testing.T) {
	srv := NewServer("test", testStore(t))
	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/analyze", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestAnalyzeCommandUnknownEventID(t *testing.T) {
	srv := NewServer("test", testStore(t))
	body := strings.NewReader(`{"event_id":"evt-does-not-exist"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/analyze", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestAnalyzeCommandLLMDisabled(t *testing.T) {
	st := testStore(t)
	srv := NewServer("test", st)
	srv.handler.LoadLLMSettings = func(string) (config.LLMSettings, error) {
		return config.LLMSettings{Enabled: false}, nil
	}

	body := strings.NewReader(`{"command":"ls -la","cwd":"/tmp"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/analyze", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestAnalyzeCommandHappyPath(t *testing.T) {
	st := testStore(t)
	srv := NewServer("test", st)
	srv.handler.LoadLLMSettings = func(string) (config.LLMSettings, error) {
		return config.LLMSettings{
			Enabled:         true,
			DefaultProvider: "test-openai",
			TimeoutMS:       3000,
			Providers: []config.ProviderInstance{
				{ID: "test-openai", Driver: "openai", Model: "gpt-4o-mini", AuthMode: "api_key"},
			},
		}, nil
	}
	srv.handler.ResolveCredentials = func() (map[string]config.ProviderCredential, error) {
		return map[string]config.ProviderCredential{
			"test-openai": {APIKey: "sk-test"},
		}, nil
	}
	srv.handler.NewAnalyzer = func(config.LLMSettings, map[string]config.ProviderCredential) (llm.Analyzer, error) {
		return &llm.MockAnalyzer{
			Result: llm.AnalyzeResult{
				Verdict:     "safe",
				Summary:     "Lists directory contents",
				Explanation: "ls is a read-only directory listing",
				Provider:    "test-openai",
			},
		}, nil
	}

	body := strings.NewReader(`{"command":"ls -la","cwd":"/tmp"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/analyze", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp AnalyzeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Verdict != "safe" {
		t.Fatalf("verdict = %q, want safe", resp.Verdict)
	}
	if resp.Summary == "" || resp.Explanation == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Provider != "test-openai" {
		t.Fatalf("provider = %q, want test-openai", resp.Provider)
	}
}

func TestAnalyzeCommandByEventID(t *testing.T) {
	st := testStore(t)
	if err := st.IngestEvent(store.CommandEvent{
		ID:              "evt-analyze-1",
		Source:          "shell",
		Client:          "cursor",
		CWD:             "/tmp/work",
		CommandRedacted: "rm -rf /tmp/foo",
		CommandNorm:     "rm -rf /tmp/foo",
		FinalAction:     "deny",
		DecisionBy:      "detect",
	}); err != nil {
		t.Fatal(err)
	}

	srv := NewServer("test", st)
	var captured llm.AnalyzeInput
	srv.handler.LoadLLMSettings = func(string) (config.LLMSettings, error) {
		return config.LLMSettings{
			Enabled:         true,
			DefaultProvider: "p1",
			TimeoutMS:       3000,
			Providers: []config.ProviderInstance{
				{ID: "p1", Driver: "openai", Model: "gpt-4o-mini", AuthMode: "api_key"},
			},
		}, nil
	}
	srv.handler.ResolveCredentials = func() (map[string]config.ProviderCredential, error) {
		return map[string]config.ProviderCredential{"p1": {APIKey: "sk-test"}}, nil
	}
	srv.handler.NewAnalyzer = func(config.LLMSettings, map[string]config.ProviderCredential) (llm.Analyzer, error) {
		return &captureAnalyzeInput{input: &captured, inner: &llm.MockAnalyzer{
			Result: llm.AnalyzeResult{
				Verdict:     "dangerous",
				Summary:     "Recursive delete",
				Explanation: "rm -rf removes files irreversibly",
				Provider:    "p1",
			},
		}}, nil
	}

	body := strings.NewReader(`{"event_id":"evt-analyze-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/analyze", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if captured.Command != "rm -rf /tmp/foo" {
		t.Fatalf("command = %q, want rm -rf /tmp/foo", captured.Command)
	}
	if captured.CWD != "/tmp/work" {
		t.Fatalf("cwd = %q, want /tmp/work", captured.CWD)
	}
}

func TestClientAnalyze(t *testing.T) {
	st := testStore(t)
	srv := NewServer("test", st)
	srv.handler.LoadLLMSettings = func(string) (config.LLMSettings, error) {
		return config.LLMSettings{
			Enabled:         true,
			DefaultProvider: "p1",
			TimeoutMS:       3000,
			Providers: []config.ProviderInstance{
				{ID: "p1", Driver: "openai", Model: "gpt-4o-mini", AuthMode: "api_key"},
			},
		}, nil
	}
	srv.handler.ResolveCredentials = func() (map[string]config.ProviderCredential, error) {
		return map[string]config.ProviderCredential{"p1": {APIKey: "sk-test"}}, nil
	}
	srv.handler.NewAnalyzer = func(config.LLMSettings, map[string]config.ProviderCredential) (llm.Analyzer, error) {
		return &llm.MockAnalyzer{
			Result: llm.AnalyzeResult{
				Verdict:     "caution",
				Summary:     "Network fetch",
				Explanation: "curl contacts a remote host",
				Provider:    "p1",
			},
		}, nil
	}

	ts := httptest.NewServer(srv.http.Handler)
	t.Cleanup(ts.Close)

	client := NewClientWithBaseURL(ts.URL)
	resp, err := client.Analyze(t.Context(), AnalyzeRequest{Command: "curl example.com", CWD: "/tmp"})
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}
	if resp.Verdict != "caution" || resp.Provider != "p1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

type captureAnalyzeInput struct {
	input *llm.AnalyzeInput
	inner llm.Analyzer
}

func (c *captureAnalyzeInput) Analyze(ctx context.Context, input llm.AnalyzeInput) (llm.AnalyzeResult, error) {
	*c.input = input
	return c.inner.Analyze(ctx, input)
}
