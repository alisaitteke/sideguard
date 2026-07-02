// Package tray panel helpers build macOS NSPopover content from poll snapshots.
// Shared row-cap and label formatters are also used by the !darwin systray menu.
// See docs/plans/2026-07-01-1537-mac-tray-popover/ (mtp-phase-3.0-darwin-panel.md).
package tray

import (
	"fmt"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalfmt"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

const (
	maxVisiblePending = 10
	maxPanelLabelLen  = 80
)

// PanelSnapshot is the poll state used to render the macOS popover panel.
type PanelSnapshot struct {
	Items    []api.PendingApproval
	Mode     approvalmode.Mode
	HealthOK bool
	Err      error
	Home     string
	Update   UpdateUIState
}

// PanelRow is one flat Allow/Deny row in the popover body.
type PanelRow struct {
	ID    string
	Label string
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

// BuildPanelRows converts a poll snapshot into header/body/footer display fields.
func BuildPanelRows(snapshot PanelSnapshot) PanelContent {
	home := snapshot.Home
	if home == "" {
		home = approvalfmt.HomeDir()
	}

	content := PanelContent{
		DaemonStatus: formatDaemonStatus(snapshot.HealthOK, snapshot.Err),
		PendingCount: formatPendingCount(len(snapshot.Items), snapshot.HealthOK),
		ModeIndex:    modeSegmentIndex(snapshot.Mode),
		ModeEnabled:  snapshot.HealthOK,
	}

	if !snapshot.HealthOK {
		content.EmptyMessage = ""
		return content
	}

	visible, overflow := visiblePendingItems(snapshot.Items, maxVisiblePending)
	for _, item := range visible {
		content.Rows = append(content.Rows, PanelRow{
			ID:    item.ID,
			Label: truncatePanelLabel(approvalfmt.FormatListLine(item, home), maxPanelLabelLen),
		})
	}

	if overflow > 0 {
		content.OverflowHint = overflowLabel(overflow)
	}

	if len(content.Rows) == 0 && content.OverflowHint == "" {
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
	return fmt.Sprintf("… and %d more (use vibeguard ui)", overflow)
}
