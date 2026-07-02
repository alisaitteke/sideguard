package tray

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
	"github.com/alisaitteke/vibeguard/internal/store"
)

func TestSessionHistory_InitialPageAndLoadMore(t *testing.T) {
	t.Parallel()

	srv, st := startTestDaemon(t)
	defer srv.Close()

	base := time.Now().UTC().Truncate(time.Second)
	total := HistoryPageSize + 5
	for i := 0; i < total; i++ {
		if err := st.IngestEvent(store.CommandEvent{
			ID:              fmt.Sprintf("evt-%03d", i),
			CreatedAt:       base.Add(time.Duration(i) * time.Second),
			Source:          "shell",
			Client:          "cursor",
			CommandRedacted: fmt.Sprintf("cmd-%d", i),
			FinalAction:     "allow",
			DecisionBy:      "user",
		}); err != nil {
			t.Fatalf("ingest event %d: %v", i, err)
		}
	}

	client := api.NewClientWithBaseURL(srv.URL)
	session := NewSession(client)

	done := make(chan struct{}, 1)
	session.OnUpdate = func(_ []api.PendingApproval, history []api.CommandEvent, _ approvalmode.Mode, historyHasMore bool, err error) {
		if err != nil {
			t.Errorf("OnUpdate err: %v", err)
		}
		if len(history) != HistoryPageSize {
			t.Errorf("initial history len = %d, want %d", len(history), HistoryPageSize)
		}
		if !historyHasMore {
			t.Error("expected historyHasMore true after initial page")
		}
		done <- struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session.Start(ctx)
	defer session.Stop()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("OnUpdate not called within timeout")
	}

	if got := len(session.History()); got != HistoryPageSize {
		t.Fatalf("History() len = %d, want %d", got, HistoryPageSize)
	}
	if !session.HistoryHasMore() {
		t.Fatal("HistoryHasMore() = false, want true")
	}

	loadCtx, loadCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer loadCancel()
	if err := session.LoadMoreHistory(loadCtx); err != nil {
		t.Fatalf("LoadMoreHistory: %v", err)
	}

	got := session.History()
	if len(got) != total {
		t.Fatalf("after load-more len = %d, want %d", len(got), total)
	}
	if session.HistoryHasMore() {
		t.Fatal("HistoryHasMore() = true, want false after partial second page")
	}

	seen := make(map[string]struct{}, len(got))
	for _, ev := range got {
		if _, dup := seen[ev.ID]; dup {
			t.Fatalf("duplicate event id %q", ev.ID)
		}
		seen[ev.ID] = struct{}{}
	}
}

func TestSessionHistory_LoadMoreThenTickPreservesExtendedHistory(t *testing.T) {
	t.Parallel()

	srv, st := startTestDaemon(t)
	defer srv.Close()

	base := time.Now().UTC().Truncate(time.Second)
	total := HistoryPageSize + 5
	for i := 0; i < total; i++ {
		if err := st.IngestEvent(store.CommandEvent{
			ID:              fmt.Sprintf("evt-%03d", i),
			CreatedAt:       base.Add(time.Duration(i) * time.Second),
			Source:          "shell",
			Client:          "cursor",
			CommandRedacted: fmt.Sprintf("cmd-%d", i),
			FinalAction:     "allow",
			DecisionBy:      "user",
		}); err != nil {
			t.Fatalf("ingest event %d: %v", i, err)
		}
	}

	client := api.NewClientWithBaseURL(srv.URL)
	session := NewSession(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session.Start(ctx)
	defer session.Stop()

	deadline := time.After(3 * time.Second)
	for len(session.History()) < HistoryPageSize {
		select {
		case <-deadline:
			t.Fatal("initial history not loaded within timeout")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	loadCtx, loadCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer loadCancel()
	if err := session.LoadMoreHistory(loadCtx); err != nil {
		t.Fatalf("LoadMoreHistory: %v", err)
	}
	if len(session.History()) != total {
		t.Fatalf("after load-more len = %d, want %d", len(session.History()), total)
	}

	session.tick(ctx)

	if got := len(session.History()); got != total {
		t.Fatalf("after tick len = %d, want %d (poll preserves loaded pages)", got, total)
	}
}
