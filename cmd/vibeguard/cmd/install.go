package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/vibeguard/internal/install"
)

var (
	installCursor      bool
	installClaude      bool
	installDryRun      bool
	installDiscover    bool
	installSkipDaemon  bool

	uninstallCursor bool
	uninstallClaude bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install VibeGuard hooks and MCP wraps for Cursor and Claude Code",
	Long:  "Discovers client configs, backs them up, wraps MCP servers with vibeguard wrap, merges hooks, and installs the daemon LaunchAgent.",
	RunE: func(_ *cobra.Command, _ []string) error {
		_, err := install.Run(install.Options{
			Cursor:     installCursor,
			Claude:     installClaude,
			DryRun:     installDryRun,
			Discover:   installDiscover,
			SkipDaemon: installSkipDaemon,
		})
		return err
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Restore client configs from the latest VibeGuard backup",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := install.Uninstall(install.Options{
			Cursor: uninstallCursor,
			Claude: uninstallClaude,
		}); err != nil {
			return err
		}
		fmt.Println("uninstall complete")
		return nil
	},
}

func init() {
	installCmd.Flags().BoolVar(&installCursor, "cursor", false, "install for Cursor only")
	installCmd.Flags().BoolVar(&installClaude, "claude", false, "install for Claude Code only")
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "print planned changes without writing files")
	installCmd.Flags().BoolVar(&installDiscover, "discover", false, "list discovered MCP servers and config paths")
	installCmd.Flags().BoolVar(&installSkipDaemon, "skip-daemon", false, "skip LaunchAgent install (hooks/MCP only)")

	uninstallCmd.Flags().BoolVar(&uninstallCursor, "cursor", false, "restore Cursor configs only")
	uninstallCmd.Flags().BoolVar(&uninstallClaude, "claude", false, "restore Claude Code configs only")

	rootCmd.AddCommand(installCmd, uninstallCmd)
}
