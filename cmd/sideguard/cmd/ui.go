// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/tui"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Interactive terminal UI for pending approvals",
	Long: `Open a keyboard-driven approval picker (arrow keys, a/d to decide).

Addresses Tier 2/3 terminal UX from docs/integration-and-terminal-ui.md.
Auto-refreshes every ~2s while running. Requires an interactive TTY.

Press g to cycle global approval mode (Ask → Auto-allow → Auto-deny).
Mode is persisted by the daemon and shared with the menu-bar tray.

YAML policy deny rules still block at the hook before items reach the queue.`,
	RunE: runUI,
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

func runUI(_ *cobra.Command, _ []string) error {
	if err := tui.Run(api.NewClient(), tui.Options{}); err != nil {
		if strings.Contains(err.Error(), "daemon unreachable") ||
			strings.Contains(err.Error(), "list pending failed") {
			return daemonNotRunningError("ui")
		}
		return err
	}
	return nil
}
