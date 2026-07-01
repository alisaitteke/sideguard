package cmd

import (
	"github.com/spf13/cobra"
	"github.com/alisaitteke/vibeguard/internal/install"
)

var (
	clientsReloadCursor bool
	clientsReloadClaude bool
)

var clientsCmd = &cobra.Command{
	Use:   "clients",
	Short: "Per-client integration helpers",
}

var clientsReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Print how to reload hooks and MCP config in Cursor and Claude Code",
	Long: `Prints honest per-client steps to pick up hook and MCP configuration changes.

VibeGuard cannot force Cursor or Claude Code to reload configuration. Use the
printed steps after install, uninstall, or manual config edits.`,
	Run: func(_ *cobra.Command, _ []string) {
		install.PrintClientReloadHints(install.Options{
			Cursor: clientsReloadCursor,
			Claude: clientsReloadClaude,
		}, "configuration changes", install.ReloadHintsFull)
	},
}

func init() {
	clientsReloadCmd.Flags().BoolVar(&clientsReloadCursor, "cursor", false, "show Cursor steps only")
	clientsReloadCmd.Flags().BoolVar(&clientsReloadClaude, "claude", false, "show Claude Code steps only")
	clientsCmd.AddCommand(clientsReloadCmd)
	rootCmd.AddCommand(clientsCmd)
}
