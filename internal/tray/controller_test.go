package tray

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
	"github.com/alisaitteke/vibeguard/internal/store"
)

func startTestDaemon(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := api.NewServer("test", st)
	return httptest.NewServer(srv.Handler()), st
}

func createPendingApproval(t *testing.T, baseURL string) string {
	t.Helper()
	body := `{"source":"hook","client":"cursor","command":"echo test"}`
	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/approval/request", strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("create approval status %d", resp.StatusCode)
	}
	var out api.ApprovalRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out.ID
}

func TestSessionPollInvokesOnUpdate(t *testing.T) {
	t.Parallel()

	srv, _ := startTestDaemon(t)
	defer srv.Close()

	id := createPendingApproval(t, srv.URL)
	client := api.NewClientWithBaseURL(srv.URL)
	ctrl := NewSession(client)

	done := make(chan struct{}, 1)
	var gotItems []api.PendingApproval
	var gotMode approvalmode.Mode
	var gotErr error

	ctrl.OnUpdate = func(items []api.PendingApproval, mode approvalmode.Mode, err error) {
		gotItems = items
		gotMode = mode
		gotErr = err
		done <- struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrl.Start(ctx)
	defer ctrl.Stop()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("OnUpdate not called within timeout")
	}

	if gotErr != nil {
		t.Fatalf("OnUpdate err = %v", gotErr)
	}
	if len(gotItems) != 1 {
		t.Fatalf("got %d items, want 1", len(gotItems))
	}
	if gotItems[0].ID != id {
		t.Fatalf("got id %q, want %q", gotItems[0].ID, id)
	}
	// Fresh stores default to auto (smart triage) since sdh Phase 3.
	if gotMode != approvalmode.Auto {
		t.Fatalf("got mode %q, want auto", gotMode)
	}
	if !ctrl.Healthy() {
		t.Fatal("expected session healthy after successful poll")
	}
}

func TestSessionPollDaemonDown(t *testing.T) {
	t.Parallel()

	client := api.NewClientWithBaseURL("http://127.0.0.1:1")
	ctrl := NewSession(client)

	done := make(chan struct{}, 1)
	var gotErr error
	ctrl.OnUpdate = func(_ []api.PendingApproval, _ approvalmode.Mode, err error) {
		gotErr = err
		done <- struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrl.Start(ctx)
	defer ctrl.Stop()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("OnUpdate not called within timeout")
	}

	if gotErr == nil {
		t.Fatal("expected daemon unreachable error")
	}
	if !strings.Contains(gotErr.Error(), "daemon unreachable") {
		t.Fatalf("got err %q, want daemon unreachable", gotErr.Error())
	}
	if ctrl.Healthy() {
		t.Fatal("expected session unhealthy when daemon is down")
	}
}

func TestSessionDecideSuccess(t *testing.T) {
	t.Parallel()

	srv, _ := startTestDaemon(t)
	defer srv.Close()

	id := createPendingApproval(t, srv.URL)
	client := api.NewClientWithBaseURL(srv.URL)
	ctrl := NewSession(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ctrl.Decide(ctx, id, "allow"); err != nil {
		t.Fatalf("Decide() error: %v", err)
	}

	pending, err := client.ListPending(ctx)
	if err != nil {
		t.Fatalf("ListPending() error: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected empty pending after allow, got %d", len(pending))
	}
}

func TestSessionDecideNotFound(t *testing.T) {
	t.Parallel()

	srv, _ := startTestDaemon(t)
	defer srv.Close()

	client := api.NewClientWithBaseURL(srv.URL)
	ctrl := NewSession(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ctrl.Decide(ctx, "00000000-0000-0000-0000-000000000000", "deny")
	if err == nil {
		t.Fatal("expected error for missing approval")
	}
	if !strings.Contains(err.Error(), "no longer pending") {
		t.Fatalf("got err %q, want user-friendly not-found message", err.Error())
	}
}
