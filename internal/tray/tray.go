// Package tray provides the menu-bar approval UI for VibeGuard.
// macOS uses a native NSPopover shell (darwin); see docs/plans/2026-07-01-1537-mac-tray-popover/.
// Other platforms use systray; see docs/plans/2026-07-01-1355-go-systray-tray/.
package tray

import (
	"fmt"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

// Options configures the menu-bar tray session.
type Options struct {
	// BaseURL overrides the daemon HTTP API base URL (default api.BaseURL()).
	BaseURL string
}

// resolveBaseURL returns the effective daemon API base URL for this tray session.
func resolveBaseURL(opts Options) string {
	if opts.BaseURL != "" {
		return opts.BaseURL
	}
	return api.BaseURL()
}

func pendingCountForTitle(items []api.PendingApproval, err error) int {
	if err != nil {
		return 0
	}
	return len(items)
}

func tooltipForUpdate(items []api.PendingApproval, mode approvalmode.Mode, err error) string {
	if err != nil {
		if strings.Contains(err.Error(), "daemon unreachable") {
			return "VibeGuard — daemon unreachable"
		}
		return "VibeGuard — " + err.Error()
	}
	suffix := ""
	switch mode {
	case approvalmode.AutoAllow:
		suffix = " — auto-allow"
	case approvalmode.AutoDeny:
		suffix = " — auto-deny"
	}
	count := len(items)
	if count == 0 {
		return "VibeGuard — no pending" + suffix
	}
	return fmt.Sprintf("VibeGuard — %d pending%s", count, suffix)
}
