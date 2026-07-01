// Package approvalfmt formats pending approval rows for CLI, TUI, and tray surfaces.
// See docs/plans/2026-07-01-1355-go-systray-tray/ (gst-phase-2.0-api-integration.md).
package approvalfmt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/api"
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

// HomeDir returns the user home directory or empty string.
func HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}
