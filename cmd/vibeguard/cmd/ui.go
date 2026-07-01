package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/tui"
)

var uiAutoApprove bool

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Interactive terminal UI for pending approvals",
	Long: `Open a keyboard-driven approval picker (arrow keys, a/d to decide).

Addresses Tier 2/3 terminal UX from docs/integration-and-terminal-ui.md.
Auto-refreshes every ~2s while running. Requires an interactive TTY.

Use --auto-approve to start in hands-off mode, or press g in the UI to toggle it on/off.
Every pending item is allowed automatically on each refresh (session-only; does not write policy rules). YAML policy deny rules
still block at the hook before items reach the queue.`,
	RunE: runUI,
}

func init() {
	rootCmd.AddCommand(uiCmd)
	uiCmd.Flags().BoolVar(&uiAutoApprove, "auto-approve", false,
		"Automatically approve all pending items (session-only; does not write policy rules)")
}

func runUI(_ *cobra.Command, _ []string) error {
	if err := tui.Run(api.NewClient(), tui.Options{AutoApprove: uiAutoApprove}); err != nil {
		if strings.Contains(err.Error(), "daemon unreachable") ||
			strings.Contains(err.Error(), "list pending failed") {
			return daemonNotRunningError("ui")
		}
		return err
	}
	return nil
}
