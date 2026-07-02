package tray

import (
	"fmt"
	"testing"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
)

func TestResolveBaseURL_default(t *testing.T) {
	got := resolveBaseURL(Options{})
	want := api.BaseURL()
	if got != want {
		t.Fatalf("resolveBaseURL() = %q, want %q", got, want)
	}
}

func TestResolveBaseURL_override(t *testing.T) {
	const override = "http://127.0.0.1:19999"
	got := resolveBaseURL(Options{BaseURL: override})
	if got != override {
		t.Fatalf("resolveBaseURL() = %q, want %q", got, override)
	}
}

func TestTooltipForUpdate(t *testing.T) {
	t.Parallel()

	if got := tooltipForUpdate(nil, approvalmode.Ask, fmt.Errorf("daemon unreachable: connection refused")); got != "SideGuard — daemon unreachable" {
		t.Fatalf("daemon down: got %q", got)
	}
	if got := tooltipForUpdate([]api.PendingApproval{{ID: "abc"}}, approvalmode.Ask, nil); got != "SideGuard — 1 pending" {
		t.Fatalf("pending: got %q", got)
	}
	if got := tooltipForUpdate(nil, approvalmode.Ask, nil); got != "SideGuard — no pending" {
		t.Fatalf("idle: got %q", got)
	}
	if got := tooltipForUpdate(nil, approvalmode.AutoAllow, nil); got != "SideGuard — no pending — auto-allow" {
		t.Fatalf("auto-allow: got %q", got)
	}
	if got := tooltipForUpdate([]api.PendingApproval{{ID: "abc"}}, approvalmode.AutoDeny, nil); got != "SideGuard — 1 pending — auto-deny" {
		t.Fatalf("auto-deny with pending: got %q", got)
	}
}
