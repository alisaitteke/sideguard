package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/vibeguard/internal/hook"
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Blocking hook bridge for Cursor and Claude Code",
	Long: `Reads JSON from stdin, submits to the VibeGuard daemon, long-polls for approval,
and writes allow/deny JSON to stdout. Used by beforeShellExecution / beforeMCPExecution
(Cursor) and PreToolUse (Claude Code).

See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-5.0-hook-bridge.md).`,
}

var hookShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Shell command hook (Cursor beforeShellExecution / Claude PreToolUse Bash)",
	Run: func(_ *cobra.Command, _ []string) {
		os.Exit(hook.RunShell(os.Stdin, os.Stdout, hook.NewClient()))
	},
}

var hookMcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP tool hook (Cursor beforeMCPExecution / Claude PreToolUse mcp__*)",
	Run: func(_ *cobra.Command, _ []string) {
		os.Exit(hook.RunMCP(os.Stdin, os.Stdout, hook.NewClient()))
	},
}

func init() {
	hookCmd.AddCommand(hookShellCmd, hookMcpCmd)
	rootCmd.AddCommand(hookCmd)
}
