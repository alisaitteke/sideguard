package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
)

func TestParseHistorySince(t *testing.T) {
	d, err := parseHistorySince("7d")
	if err != nil {
		t.Fatal(err)
	}
	if d != 7*24*time.Hour {
		t.Fatalf("7d = %v", d)
	}
	d, err = parseHistorySince("24h")
	if err != nil || d != 24*time.Hour {
		t.Fatalf("24h = %v err=%v", d, err)
	}
}

func TestTruncateHistoryCommand(t *testing.T) {
	short := "git status"
	if got := truncateHistoryCommand(short); got != short {
		t.Fatalf("short command changed: %q", got)
	}
	long := strings.Repeat("a", 80)
	got := truncateHistoryCommand(long)
	if len(got) != historyMaxDisplay {
		t.Fatalf("len = %d, want %d", len(got), historyMaxDisplay)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis suffix: %q", got)
	}
}

func TestQueryHistoryMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/events" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("denied"); got != "true" {
			t.Fatalf("denied = %q, want true", got)
		}
		if got := r.URL.Query().Get("search"); got != "rm" {
			t.Fatalf("search = %q, want rm", got)
		}
		if r.URL.Query().Get("since") == "" {
			t.Fatal("expected since query param")
		}
		_ = json.NewEncoder(w).Encode([]api.CommandEvent{
			{
				CreatedAt:       time.Now().UTC().Format(time.RFC3339),
				FinalAction:     "deny",
				DecisionBy:      "detect",
				CommandRedacted: "rm -rf /",
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := api.NewClientWithBaseURL(srv.URL)
	events, err := queryHistory(context.Background(), client, historyQueryOpts{
		Since:  time.Now().UTC().Add(-time.Hour),
		Denied: true,
		Search: "rm",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].FinalAction != "deny" {
		t.Fatalf("events = %+v", events)
	}
}

func TestQueryHistoryDaemonDown(t *testing.T) {
	client := api.NewClientWithBaseURL("http://127.0.0.1:1")
	_, err := queryHistory(context.Background(), client, historyQueryOpts{
		Since: time.Now().UTC().Add(-time.Hour),
	})
	if err == nil || !strings.Contains(err.Error(), "daemon unreachable") {
		t.Fatalf("err = %v, want daemon unreachable", err)
	}
}
