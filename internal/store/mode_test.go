package store

import (
	"testing"

	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

func TestGetApprovalModeDefault(t *testing.T) {
	s := openTestDB(t)
	mode, err := s.GetApprovalMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != approvalmode.Auto {
		t.Fatalf("mode = %q, want auto", mode)
	}
}

func TestSetGetApprovalModeAuto(t *testing.T) {
	s := openTestDB(t)
	if err := s.SetApprovalMode(approvalmode.Auto); err != nil {
		t.Fatal(err)
	}
	mode, err := s.GetApprovalMode()
	if err != nil || mode != approvalmode.Auto {
		t.Fatalf("got %q err=%v", mode, err)
	}
}

func TestSetGetApprovalModeRoundtrip(t *testing.T) {
	s := openTestDB(t)
	if err := s.SetApprovalMode(approvalmode.AutoAllow); err != nil {
		t.Fatal(err)
	}
	mode, err := s.GetApprovalMode()
	if err != nil || mode != approvalmode.AutoAllow {
		t.Fatalf("got %q err=%v", mode, err)
	}
}

func TestMaybeAutoDecideAllow(t *testing.T) {
	s := openTestDB(t)
	if err := s.SetApprovalMode(approvalmode.AutoAllow); err != nil {
		t.Fatal(err)
	}

	rec, err := s.CreateApproval("shell", "cursor", "echo", "/tmp", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	result, decided, err := s.MaybeAutoDecide(rec.ID)
	if err != nil || !decided {
		t.Fatalf("MaybeAutoDecide: decided=%v err=%v", decided, err)
	}
	if result.Permission != "allow" {
		t.Fatalf("permission = %q", result.Permission)
	}

	pending, err := s.ListPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending count = %d, want 0", len(pending))
	}
}

func TestMaybeAutoDecideAskNoOp(t *testing.T) {
	s := openTestDB(t)
	rec, err := s.CreateApproval("shell", "cursor", "echo", "/tmp", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, decided, err := s.MaybeAutoDecide(rec.ID)
	if err != nil || decided {
		t.Fatalf("ask mode: decided=%v err=%v", decided, err)
	}

	pending, err := s.ListPending()
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %d err=%v", len(pending), err)
	}
}

func TestSetApprovalModeSweepsPending(t *testing.T) {
	s := openTestDB(t)
	r1, err := s.CreateApproval("shell", "cursor", "a", "/tmp", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := s.CreateApproval("shell", "cursor", "b", "/tmp", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SetApprovalMode(approvalmode.AutoDeny); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{r1.ID, r2.ID} {
		rec, err := s.GetApproval(id)
		if err != nil {
			t.Fatal(err)
		}
		if rec.Decision != "deny" {
			t.Fatalf("%s decision = %q, want deny", id, rec.Decision)
		}
	}

	pending, err := s.ListPending()
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending = %d err=%v", len(pending), err)
	}
}

func TestSetApprovalModeWritesAudit(t *testing.T) {
	s := openTestDB(t)
	if err := s.SetApprovalMode(approvalmode.AutoAllow); err != nil {
		t.Fatal(err)
	}

	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM audit_events WHERE event_type = 'approval_mode_changed'`).Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("mode_changed audit count = %d err=%v", count, err)
	}
}
