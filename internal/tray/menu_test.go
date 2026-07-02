package tray

import (
	"strings"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

func TestFormatModeStatus(t *testing.T) {
	t.Parallel()

	if got := formatModeStatus(approvalmode.Ask, true); got != "● Mode: Ask" {
		t.Fatalf("ask: %q", got)
	}
	if got := formatModeStatus(approvalmode.AutoAllow, true); got != "● Mode: Auto-allow" {
		t.Fatalf("auto-allow: %q", got)
	}
	if got := formatModeStatus(approvalmode.AutoDeny, false); got != "● Mode: unavailable" {
		t.Fatalf("down: %q", got)
	}
}

func TestTruncateMenuLabel(t *testing.T) {
	t.Parallel()

	short := "abc"
	if got := truncateMenuLabel(short, 80); got != short {
		t.Fatalf("short label: got %q", got)
	}

	long := strings.Repeat("x", 100)
	got := truncateMenuLabel(long, 80)
	if len([]rune(got)) != 80 {
		t.Fatalf("truncated rune len = %d, want 80", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis suffix, got %q", got)
	}
}

func TestVisiblePendingItems(t *testing.T) {
	t.Parallel()

	items := make([]api.PendingApproval, 12)
	for i := range items {
		items[i].ID = "id"
	}

	visible, overflow := visiblePendingItems(items, maxVisiblePending)
	if len(visible) != maxVisiblePending {
		t.Fatalf("visible = %d, want %d", len(visible), maxVisiblePending)
	}
	if overflow != 2 {
		t.Fatalf("overflow = %d, want 2", overflow)
	}

	visible, overflow = visiblePendingItems(items[:3], maxVisiblePending)
	if len(visible) != 3 || overflow != 0 {
		t.Fatalf("under cap: visible=%d overflow=%d", len(visible), overflow)
	}

	visible, overflow = visiblePendingItems(nil, maxVisiblePending)
	if len(visible) != 0 || overflow != 0 {
		t.Fatalf("empty: visible=%d overflow=%d", len(visible), overflow)
	}
}

func TestOverflowLabel(t *testing.T) {
	t.Parallel()

	got := overflowLabel(2)
	want := "… and 2 more (use vibeguard ui)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatDaemonStatus(t *testing.T) {
	t.Parallel()

	if got := formatDaemonStatus(true, nil); got != "● Daemon: OK" {
		t.Fatalf("healthy: %q", got)
	}
	if got := formatDaemonStatus(false, errDaemonUnreachable()); got != "● Daemon: unreachable" {
		t.Fatalf("unreachable: %q", got)
	}
}

func TestFormatPendingCount(t *testing.T) {
	t.Parallel()

	if got := formatPendingCount(0, true); got != "● 0 pending" {
		t.Fatalf("zero: %q", got)
	}
	if got := formatPendingCount(2, true); got != "● 2 pending" {
		t.Fatalf("two: %q", got)
	}
	if got := formatPendingCount(1, false); got != "● pending unavailable" {
		t.Fatalf("down: %q", got)
	}
}

func TestVisibleSystrayHistory(t *testing.T) {
	t.Parallel()

	row := func(id string) TrayRow {
		return TrayRow{Kind: TrayRowHistory, ID: id, Label: id}
	}

	rows := make([]TrayRow, 20)
	for i := range rows {
		rows[i] = row("evt")
	}

	visible, showSection, showLoadMore := visibleSystrayHistory(rows, true, maxVisibleHistory)
	if !showSection {
		t.Fatal("expected history section visible")
	}
	if len(visible) != maxVisibleHistory {
		t.Fatalf("visible = %d, want %d", len(visible), maxVisibleHistory)
	}
	if !showLoadMore {
		t.Fatal("expected load-more visible when hasMore")
	}

	visible, showSection, showLoadMore = visibleSystrayHistory(rows[:3], false, maxVisibleHistory)
	if !showSection || len(visible) != 3 || showLoadMore {
		t.Fatalf("small history: section=%v visible=%d loadMore=%v", showSection, len(visible), showLoadMore)
	}

	_, showSection, showLoadMore = visibleSystrayHistory(nil, false, maxVisibleHistory)
	if showSection || showLoadMore {
		t.Fatalf("empty history: section=%v loadMore=%v", showSection, showLoadMore)
	}
}

func errDaemonUnreachable() error {
	return &daemonUnreachableErr{}
}

type daemonUnreachableErr struct{}

func (e *daemonUnreachableErr) Error() string {
	return "daemon unreachable: connection refused"
}
