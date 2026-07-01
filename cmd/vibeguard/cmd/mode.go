package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

var modeCmd = &cobra.Command{
	Use:   "mode",
	Short: "Show or set global approval mode",
	Long: `Global approval mode is persisted by the daemon and shared by tray and TUI.

Modes:
  ask         Manual Allow/Deny for each queued request (default)
  auto-allow  Auto-allow every queued request (audit logged)
  auto-deny   Auto-deny every queued request (audit logged)

Auto modes apply to existing pending items when set.`,
	RunE: runMode,
}

var modeSetCmd = &cobra.Command{
	Use:   "set <ask|auto-allow|auto-deny>",
	Short: "Set global approval mode",
	Args:  cobra.ExactArgs(1),
	RunE:  runModeSet,
}

func init() {
	modeCmd.AddCommand(modeSetCmd)
	rootCmd.AddCommand(modeCmd)
}

func runMode(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	mode, err := api.NewClient().GetApprovalMode(ctx)
	if err != nil {
		return daemonNotRunningError("mode")
	}
	fmt.Fprintln(os.Stdout, mode.Label())
	return nil
}

func runModeSet(_ *cobra.Command, args []string) error {
	mode, err := approvalmode.ParseCLI(args[0])
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := api.NewClient().SetApprovalMode(ctx, mode); err != nil {
		return daemonNotRunningError("mode set")
	}
	fmt.Fprintf(os.Stdout, "approval mode set to %s\n", mode.Label())
	return nil
}
