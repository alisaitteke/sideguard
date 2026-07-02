package cmd

import (
	"errors"
	"testing"

	"github.com/alisaitteke/sideguard/internal/api"
)

func TestResolveApprovalID_emptyID_noPending(t *testing.T) {
	_, err := resolveApprovalID("", nil)
	if !errors.Is(err, ErrNoPendingApprovals) {
		t.Fatalf("err = %v, want ErrNoPendingApprovals", err)
	}
}

func TestResolveApprovalID_emptyID_onePending(t *testing.T) {
	pending := []api.PendingApproval{{ID: "abc123def456"}}
	id, err := resolveApprovalID("", pending)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != "abc123def456" {
		t.Fatalf("id = %q, want abc123def456", id)
	}
}

func TestResolveApprovalID_emptyID_manyPending(t *testing.T) {
	pending := []api.PendingApproval{
		{ID: "id-one"},
		{ID: "id-two"},
	}
	_, err := resolveApprovalID("", pending)
	var multi *MultiplePendingError
	if !errors.As(err, &multi) {
		t.Fatalf("err = %T %v, want *MultiplePendingError", err, err)
	}
	if len(multi.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(multi.Items))
	}
}

func TestResolveApprovalID_exactMatch(t *testing.T) {
	pending := []api.PendingApproval{
		{ID: "full-id-aaa"},
		{ID: "full-id-bbb"},
	}
	id, err := resolveApprovalID("full-id-bbb", pending)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != "full-id-bbb" {
		t.Fatalf("id = %q, want full-id-bbb", id)
	}
}

func TestResolveApprovalID_prefixMatch(t *testing.T) {
	pending := []api.PendingApproval{{ID: "abcdef123456"}}
	id, err := resolveApprovalID("abcdef", pending)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != "abcdef123456" {
		t.Fatalf("id = %q, want full id", id)
	}
}

func TestResolveApprovalID_unknownID_passThrough(t *testing.T) {
	pending := []api.PendingApproval{{ID: "other-id"}}
	id, err := resolveApprovalID("not-in-queue", pending)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != "not-in-queue" {
		t.Fatalf("id = %q, want pass-through", id)
	}
}

func TestResolveApprovalID_whitespaceID(t *testing.T) {
	pending := []api.PendingApproval{{ID: "solo-id"}}
	id, err := resolveApprovalID("   ", pending)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != "solo-id" {
		t.Fatalf("id = %q, want solo-id", id)
	}
}
