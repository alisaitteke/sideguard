package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateAndDecideApproval(t *testing.T) {
	s := openTestDB(t)

	rec, err := s.CreateApproval("shell", "cursor", "echo test", "/tmp", "", nil)
	if err != nil {
		t.Fatalf("CreateApproval: %v", err)
	}
	if rec.Status != "pending" {
		t.Fatalf("status = %q, want pending", rec.Status)
	}

	result, changed, err := s.DecideApproval(rec.ID, "allow", "ok")
	if err != nil || !changed {
		t.Fatalf("DecideApproval: changed=%v err=%v", changed, err)
	}
	if result.Permission != "allow" {
		t.Fatalf("permission = %q", result.Permission)
	}

	again, changed, err := s.DecideApproval(rec.ID, "allow", "ok")
	if err != nil || changed {
		t.Fatalf("idempotent decide: changed=%v err=%v", changed, err)
	}
	if again.Permission != "allow" {
		t.Fatalf("permission = %q", again.Permission)
	}
}

func TestWaitForDecision(t *testing.T) {
	s := openTestDB(t)
	rec, err := s.CreateApproval("shell", "cursor", "rm -rf /", "/tmp", "", nil)
	if err != nil {
		t.Fatalf("CreateApproval: %v", err)
	}

	done := make(chan *DecisionResult, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result, err := s.WaitForDecision(ctx, rec.ID)
		if err != nil {
			t.Errorf("WaitForDecision: %v", err)
			done <- nil
			return
		}
		done <- result
	}()

	time.Sleep(50 * time.Millisecond)
	if _, _, err := s.DecideApproval(rec.ID, "deny", "too dangerous"); err != nil {
		t.Fatalf("DecideApproval: %v", err)
	}

	select {
	case result := <-done:
		if result == nil || result.Permission != "deny" {
			t.Fatalf("unexpected wait result: %+v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait timed out")
	}
}

func TestListPending(t *testing.T) {
	s := openTestDB(t)
	if _, err := s.CreateApproval("shell", "cursor", "a", "/tmp", "", nil); err != nil {
		t.Fatal(err)
	}
	pending, err := s.ListPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending count = %d, want 1", len(pending))
	}
}

func openTestDB(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
