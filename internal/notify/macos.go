//go:build darwin

package notify

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// sendMacOS delivers an alert-only notification on macOS.
// Prefers terminal-notifier when installed; falls back to osascript.
// See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-6.0-terminal-ui.md).
func sendMacOS(title, body string) error {
	title = truncate(title, maxTitleLen)
	body = truncate(body, maxBodyLen)

	if path, err := findTerminalNotifier(); err == nil {
		cmd := exec.Command(path,
			"-title", title,
			"-message", body,
			"-sender", "com.apple.Terminal",
		)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	script := fmt.Sprintf(
		`display notification %q with title %q`,
		body, title,
	)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("osascript notification: %w", err)
	}
	return nil
}

func findTerminalNotifier() (string, error) {
	candidates := []string{
		"/opt/homebrew/bin/terminal-notifier",
		"/usr/local/bin/terminal-notifier",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return exec.LookPath("terminal-notifier")
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
