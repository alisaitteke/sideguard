// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Package approvalfmt formats pending approval rows for CLI, TUI, and tray surfaces.
// See docs/plans/2026-07-01-1355-go-systray-tray/ (gst-phase-2.0-api-integration.md).
package approvalfmt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alisaitteke/sideguard/internal/api"
)

// ShortApprovalID returns a short display id like "#a1b2c3d4".
func ShortApprovalID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > 8 {
		return "#" + id[:8]
	}
	if id == "" {
		return "#?"
	}
	return "#" + id
}

// FormatClientLabel normalizes the client name for display.
func FormatClientLabel(client string) string {
	client = strings.TrimSpace(strings.ToLower(client))
	switch client {
	case "cursor":
		return "cursor"
	case "claude":
		return "claude"
	case "":
		return "agent"
	default:
		return client
	}
}

// FormatAgeLong returns a human-readable age like "5s ago".
func FormatAgeLong(seconds int64) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%ds ago", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm ago", seconds/60)
	default:
		return fmt.Sprintf("%dh ago", seconds/3600)
	}
}

// FormatAgeShort returns a compact age like "5s" for list rows.
func FormatAgeShort(seconds int64) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%ds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm", seconds/60)
	default:
		return fmt.Sprintf("%dh", seconds/3600)
	}
}

// FormatCWD shortens cwd for display (~ prefix when under home).
func FormatCWD(cwd, home string) string {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return "."
	}
	if home != "" {
		if cwd == home {
			return "~"
		}
		prefix := home + string(filepath.Separator)
		if strings.HasPrefix(cwd, prefix) {
			return "~" + string(filepath.Separator) + strings.TrimPrefix(cwd, prefix)
		}
	}
	return cwd
}

// FormatSummary returns command or tool summary for a pending approval.
func FormatSummary(item api.PendingApproval) string {
	if strings.TrimSpace(item.Command) != "" {
		return item.Command
	}
	if strings.TrimSpace(item.ToolName) != "" {
		if item.Source == "mcp" {
			return "mcp:" + item.ToolName
		}
		return item.ToolName
	}
	return "(no command detail)"
}

// FormatTrayRowLabel returns command-only text for tray list rows (no id/client/age).
func FormatTrayRowLabel(item api.PendingApproval) string {
	return FormatSummary(item)
}

// FormatTrayEventLabel returns command-only text for tray history rows.
func FormatTrayEventLabel(ev api.CommandEvent) string {
	return FormatEventSummary(ev)
}

// FormatListLine builds the main UI row: "#id · client · age · summary".
func FormatListLine(item api.PendingApproval, home string) string {
	_ = home
	return fmt.Sprintf("%s · %s · %s · %s",
		ShortApprovalID(item.ID),
		FormatClientLabel(item.Client),
		FormatAgeShort(item.AgeSeconds),
		FormatSummary(item),
	)
}

// FormatTrayPendingLine builds a compact tray row without the approval id prefix.
func FormatTrayPendingLine(item api.PendingApproval, home string) string {
	_ = home
	return fmt.Sprintf("%s · %s · %s",
		FormatClientLabel(item.Client),
		FormatAgeShort(item.AgeSeconds),
		FormatSummary(item),
	)
}

// FormatEventSummary returns command or tool summary for a history event.
func FormatEventSummary(ev api.CommandEvent) string {
	if strings.TrimSpace(ev.CommandRedacted) != "" {
		return ev.CommandRedacted
	}
	if strings.TrimSpace(ev.ToolName) != "" {
		if ev.Source == "mcp" {
			return "mcp:" + ev.ToolName
		}
		return ev.ToolName
	}
	return "(no command detail)"
}

// FormatEventAge returns a compact age string from an RFC3339 created_at timestamp.
func FormatEventAge(createdAt string) string {
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return "?"
	}
	secs := int64(time.Since(t).Seconds())
	if secs < 0 {
		secs = 0
	}
	return FormatAgeShort(secs)
}

// FormatEventLine builds a read-only history row:
// "#id · client · age · action · summary".
// See docs/plans/2026-07-02-1226-tray-ui-polish/ (tup-phase-2.0-tray-core.md).
func FormatEventLine(ev api.CommandEvent, home string) string {
	_ = home
	id := ShortApprovalID(ev.ApprovalID)
	if strings.TrimSpace(ev.ApprovalID) == "" {
		id = ShortApprovalID(ev.ID)
	}
	action := strings.TrimSpace(ev.FinalAction)
	if action == "" {
		action = "?"
	}
	return fmt.Sprintf("%s · %s · %s · %s · %s",
		id,
		FormatClientLabel(ev.Client),
		FormatEventAge(ev.CreatedAt),
		action,
		FormatEventSummary(ev),
	)
}

// FormatTrayHistoryLine builds a compact tray history row without id or action text.
func FormatTrayHistoryLine(ev api.CommandEvent, home string) string {
	_ = home
	return fmt.Sprintf("%s · %s · %s",
		FormatClientLabel(ev.Client),
		FormatEventAge(ev.CreatedAt),
		FormatEventSummary(ev),
	)
}

// HomeDir returns the user home directory or empty string.
func HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}
