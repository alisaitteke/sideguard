//go:build darwin

package darwin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alisaitteke/sideguard/internal/api"
)

func TestBuildAnalyzeRequestEventID(t *testing.T) {
	req := BuildAnalyzeRequest("evt-42", "ignored command", true)
	if req.EventID != "evt-42" {
		t.Fatalf("EventID = %q, want evt-42", req.EventID)
	}
	if req.Command != "" {
		t.Fatalf("Command = %q, want empty when event_id is used", req.Command)
	}
}

func TestBuildAnalyzeRequestCommandOnly(t *testing.T) {
	req := BuildAnalyzeRequest("approval-1", "rm -rf /", false)
	if req.EventID != "" {
		t.Fatalf("EventID = %q, want empty for pending row", req.EventID)
	}
	if req.Command != "rm -rf /" {
		t.Fatalf("Command = %q, want rm -rf /", req.Command)
	}
}

func TestBuildAnalyzeRequestEmptyEventFallsBackToCommand(t *testing.T) {
	req := BuildAnalyzeRequest("", "ls -la", true)
	if req.EventID != "" {
		t.Fatalf("EventID = %q, want empty", req.EventID)
	}
	if req.Command != "ls -la" {
		t.Fatalf("Command = %q, want ls -la", req.Command)
	}
}

func TestSanitizeAnalyseErrorDaemonUnreachable(t *testing.T) {
	msg := SanitizeAnalyseError(fmt.Errorf("daemon unreachable: %w", errors.New("connection refused")))
	if msg == "" || !strings.Contains(msg, "Daemon unreachable") {
		t.Fatalf("message = %q, want daemon unreachable hint", msg)
	}
}

func TestSanitizeAnalyseErrorNoProvider(t *testing.T) {
	err := fmt.Errorf("analyze failed: status 503: no llm provider configured")
	msg := SanitizeAnalyseError(err)
	if !strings.Contains(msg, "Settings") {
		t.Fatalf("message = %q, want settings hint", msg)
	}
}

func TestRunAnalyzeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/analyze" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req api.AnalyzeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.EventID != "evt-1" {
			t.Fatalf("EventID = %q, want evt-1", req.EventID)
		}
		_ = json.NewEncoder(w).Encode(api.AnalyzeResponse{
			Verdict:     "caution",
			Summary:     "Downloads remote content",
			Explanation: "curl may fetch untrusted scripts",
			Provider:    "test-openai",
		})
	}))
	t.Cleanup(srv.Close)

	client := api.NewClientWithBaseURL(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := RunAnalyze(ctx, client, "evt-1", "ignored", true)
	if result.Error != "" {
		t.Fatalf("Error = %q, want empty", result.Error)
	}
	if result.Verdict != "caution" {
		t.Fatalf("Verdict = %q, want caution", result.Verdict)
	}
	if result.Summary == "" || result.Explanation == "" {
		t.Fatalf("missing summary/explanation: %+v", result)
	}
}

func TestRunAnalyzeDaemonError(t *testing.T) {
	client := api.NewClientWithBaseURL("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result := RunAnalyze(ctx, client, "", "ls", false)
	if result.Error == "" {
		t.Fatal("expected error for unreachable daemon")
	}
	if !strings.Contains(result.Error, "Daemon unreachable") {
		t.Fatalf("Error = %q, want daemon unreachable message", result.Error)
	}
}

func TestRunAnalyzeEmptyInput(t *testing.T) {
	client := api.NewClientWithBaseURL("http://127.0.0.1:1")
	result := RunAnalyze(context.Background(), client, "", "", false)
	if result.Error != "No command to analyze" {
		t.Fatalf("Error = %q, want no command message", result.Error)
	}
}
