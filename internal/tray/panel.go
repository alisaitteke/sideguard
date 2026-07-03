// Package tray panel helpers build macOS NSPopover content from poll snapshots.
// Shared row-cap and label formatters are also used by the !darwin systray menu.
// See docs/plans/2026-07-01-1537-mac-tray-popover/ (mtp-phase-3.0-darwin-panel.md).
package tray

import (
	"fmt"
	"strings"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalfmt"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
)

const (
	maxVisiblePending = 10
	maxPanelLabelLen  = 80
	// maxVisibleHistory caps systray history rows; darwin scroll shows all loaded rows.
	maxVisibleHistory = 15
)

// PanelSnapshot is the poll state used to render tray content (pending + history).
type PanelSnapshot struct {
	Items          []api.PendingApproval
	History        []api.CommandEvent
	HistoryHasMore bool
	Mode           approvalmode.Mode
	HealthOK       bool
	Err            error
	Home           string
	Update         UpdateUIState
}

// PanelRow is one flat Run/Decline row in the popover body (pending block).
type PanelRow struct {
	ID     string
	Label  string
	Detail string
	Kind   TrayRowKind
}

// TrayRowKind distinguishes pending approval rows from read-only history rows.
type TrayRowKind string

const (
	TrayRowPending TrayRowKind = "pending"
	TrayRowHistory TrayRowKind = "history"
)

// TrayRow is one row in the merged tray list (pending or history).
type TrayRow struct {
	Kind   TrayRowKind
	ID     string
	Label  string // truncated command for the list row
	Detail string // full command shown when the row is clicked
}

// TrayContent is the display-ready merged tray payload for darwin and systray.
// See docs/plans/2026-07-02-1226-tray-ui-polish/ (tup-phase-2.0-tray-core.md).
type TrayContent struct {
	FooterDaemon    string
	FooterPending   string
	ModeIndex       int
	ModeEnabled     bool
	PendingRows     []TrayRow
	HistoryRows     []TrayRow
	HistoryHasMore  bool
	PendingOverflow string
	EmptyMessage    string
	UpdateLabel     string
	UpdateVisible   bool
	UpdateEnabled   bool
}

// PanelContent is the display-ready panel payload for darwin.UpdatePanel.
type PanelContent struct {
	DaemonStatus  string
	PendingCount  string
	ModeIndex     int
	ModeEnabled   bool
	Rows          []PanelRow
	OverflowHint  string
	EmptyMessage  string
	UpdateLabel   string
	UpdateVisible bool
	UpdateEnabled bool
}

// BuildTrayContent merges pending and history blocks for tray UI backends.
// Pending rows keep API order (newest first); history excludes rows still pending.
func BuildTrayContent(snapshot PanelSnapshot) TrayContent {
	home := snapshot.Home
	if home == "" {
		home = approvalfmt.HomeDir()
	}

	content := TrayContent{
		FooterDaemon:  formatDaemonStatus(snapshot.HealthOK, snapshot.Err),
		FooterPending: formatPendingCount(len(snapshot.Items), snapshot.HealthOK),
		ModeIndex:     modeSegmentIndex(snapshot.Mode),
		ModeEnabled:   snapshot.HealthOK,
	}

	if !snapshot.HealthOK {
		return content
	}

	pendingIDs := make(map[string]struct{}, len(snapshot.Items))
	for _, item := range snapshot.Items {
		pendingIDs[item.ID] = struct{}{}
	}

	// Darwin carousel shows every pending item; systray caps rows via menu slots + overflow hint.
	for _, item := range snapshot.Items {
		summary := approvalfmt.FormatTrayRowLabel(item)
		content.PendingRows = append(content.PendingRows, TrayRow{
			Kind:   TrayRowPending,
			ID:     item.ID,
			Label:  truncatePanelLabel(summary, maxPanelLabelLen),
			Detail: summary,
		})
	}
	if overflow := len(snapshot.Items) - maxVisiblePending; overflow > 0 {
		content.PendingOverflow = overflowLabel(overflow)
	}

	for _, ev := range snapshot.History {
		if aid := strings.TrimSpace(ev.ApprovalID); aid != "" {
			if _, stillPending := pendingIDs[aid]; stillPending {
				continue
			}
		}
		summary := approvalfmt.FormatTrayEventLabel(ev)
		content.HistoryRows = append(content.HistoryRows, TrayRow{
			Kind:   TrayRowHistory,
			ID:     ev.ID,
			Label:  truncatePanelLabel(summary, maxPanelLabelLen),
			Detail: summary,
		})
	}
	content.HistoryHasMore = snapshot.HistoryHasMore

	if len(content.PendingRows) == 0 && len(content.HistoryRows) == 0 && content.PendingOverflow == "" {
		content.EmptyMessage = "No pending approvals"
	}

	if snapshot.Update.Available {
		content.UpdateVisible = true
		content.UpdateLabel = snapshot.Update.Label
		content.UpdateEnabled = !snapshot.Update.Installing
		if content.UpdateLabel == "" && snapshot.Update.Version != "" {
			content.UpdateLabel = fmt.Sprintf("Update available: v%s", snapshot.Update.Version)
		}
	}

	return content
}

// BuildPanelRows converts a poll snapshot into header/body/footer display fields.
// Pending rows only — full merged content is BuildTrayContent (Phase 3 darwin uses TrayContent).
func BuildPanelRows(snapshot PanelSnapshot) PanelContent {
	tray := BuildTrayContent(snapshot)
	content := PanelContent{
		DaemonStatus:  tray.FooterDaemon,
		PendingCount:  tray.FooterPending,
		ModeIndex:     tray.ModeIndex,
		ModeEnabled:   tray.ModeEnabled,
		OverflowHint:  tray.PendingOverflow,
		EmptyMessage:  tray.EmptyMessage,
		UpdateLabel:   tray.UpdateLabel,
		UpdateVisible: tray.UpdateVisible,
		UpdateEnabled: tray.UpdateEnabled,
	}
	visible, overflow := visiblePendingItems(snapshot.Items, maxVisiblePending)
	if overflow > 0 {
		content.OverflowHint = overflowLabel(overflow)
	}
	if snapshot.HealthOK {
		for _, item := range visible {
			summary := approvalfmt.FormatTrayRowLabel(item)
			content.Rows = append(content.Rows, PanelRow{
				ID:     item.ID,
				Label:  truncatePanelLabel(summary, maxPanelLabelLen),
				Detail: summary,
				Kind:   TrayRowPending,
			})
		}
	}
	return content
}

// Segment order matches the NSSegmentedControl labels in darwin/status_popover.m:
// 0 Ask · 1 Auto (smart triage) · 2 Auto-allow · 3 Auto-deny.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-11.0-tray-tui.md).
func modeSegmentIndex(mode approvalmode.Mode) int {
	switch mode {
	case approvalmode.Auto:
		return 1
	case approvalmode.AutoAllow:
		return 2
	case approvalmode.AutoDeny:
		return 3
	default:
		return 0
	}
}

// ModeFromSegmentIndex maps NSSegmentedControl index to approval mode.
func ModeFromSegmentIndex(index int) approvalmode.Mode {
	switch index {
	case 1:
		return approvalmode.Auto
	case 2:
		return approvalmode.AutoAllow
	case 3:
		return approvalmode.AutoDeny
	default:
		return approvalmode.Ask
	}
}

// formatModeStatus returns the disabled status row label for the current mode.
func formatModeStatus(mode approvalmode.Mode, healthOK bool) string {
	if !healthOK {
		return "● Mode: unavailable"
	}
	return "● Mode: " + mode.Label()
}

// truncatePanelLabel shortens a row label for native UI (rune-aware).
func truncatePanelLabel(label string, maxLen int) string {
	runes := []rune(label)
	if maxLen <= 0 || len(runes) <= maxLen {
		return label
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// truncateMenuLabel is an alias for systray menu rows.
func truncateMenuLabel(label string, maxLen int) string {
	return truncatePanelLabel(label, maxLen)
}

// visiblePendingItems returns up to cap items and how many remain hidden.
func visiblePendingItems(items []api.PendingApproval, cap int) (visible []api.PendingApproval, overflow int) {
	if cap <= 0 || len(items) == 0 {
		if len(items) > 0 {
			return nil, len(items)
		}
		return nil, 0
	}
	if len(items) <= cap {
		return items, 0
	}
	return items[:cap], len(items) - cap
}

func formatDaemonStatus(healthOK bool, err error) string {
	if err != nil || !healthOK {
		if err != nil && strings.Contains(err.Error(), "daemon unreachable") {
			return "● Daemon: unreachable"
		}
		if err != nil {
			return "● Daemon: " + err.Error()
		}
		return "● Daemon: unreachable"
	}
	return "● Daemon: OK"
}

func formatPendingCount(n int, healthOK bool) string {
	if !healthOK {
		return "● pending unavailable"
	}
	switch n {
	case 0:
		return "● 0 pending"
	case 1:
		return "● 1 pending"
	default:
		return fmt.Sprintf("● %d pending", n)
	}
}

func overflowLabel(overflow int) string {
	return fmt.Sprintf("… and %d more (use sideguard ui)", overflow)
}

// visibleSystrayHistory returns capped history rows and whether the systray section / load-more show.
// See docs/plans/2026-07-02-1226-tray-ui-polish/ (tup-phase-4.0-tray-systray.md).
func visibleSystrayHistory(rows []TrayRow, hasMore bool, cap int) (visible []TrayRow, showSection, showLoadMore bool) {
	if len(rows) == 0 {
		return nil, false, false
	}
	showSection = true
	if cap <= 0 || len(rows) <= cap {
		visible = rows
	} else {
		visible = rows[:cap]
	}
	showLoadMore = hasMore
	return visible, showSection, showLoadMore
}
