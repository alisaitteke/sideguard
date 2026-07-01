//go:build !darwin

package tray

import (
	"github.com/getlantern/systray"
)

// Run starts the menu-bar tray and blocks until the user chooses Quit.
// Requires CGO_ENABLED=1 and an active GUI session (macOS menu bar).
// The tray stays visible even when the daemon is unreachable; tooltip reflects poll state.
func Run(opts Options) error {
	baseURL := resolveBaseURL(opts)
	session := &systraySession{baseURL: baseURL}

	systray.Run(session.onReady, session.onExit)
	return nil
}
