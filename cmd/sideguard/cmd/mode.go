package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
	"github.com/spf13/cobra"
)

var modeCmd = &cobra.Command{
	Use:   "mode",
	Short: "Show or set global approval mode",
	Long: `Global approval mode is persisted by the daemon and shared by tray and TUI.

Modes:
  ask         Manual Allow/Deny for each queued request
  auto        Smart triage: safe commands pass, risky ones blocked, uncertain queued
  auto-allow  Auto-allow every queued request (audit logged)
  auto-deny   Auto-deny every queued request (audit logged)

Auto-allow and auto-deny apply to existing pending items when set.`,
	RunE: runMode,
}

var modeSetCmd = &cobra.Command{
	Use:   "set <ask|auto|auto-allow|auto-deny>",
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
		if strings.Contains(err.Error(), "daemon unreachable") {
			return daemonNotRunningError("mode set")
		}
		return fmt.Errorf("mode set failed: %w", err)
	}
	fmt.Fprintf(os.Stdout, "approval mode set to %s\n", mode.Label())
	return nil
}
