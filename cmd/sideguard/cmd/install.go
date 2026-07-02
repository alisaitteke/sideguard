package cmd

import (
	"github.com/spf13/cobra"
	"github.com/alisaitteke/sideguard/internal/install"
)

var (
	installCursor     bool
	installClaude     bool
	installDryRun     bool
	installDiscover   bool
	installSkipDaemon bool
	installHeadless   bool
	installDev        bool

	uninstallCursor        bool
	uninstallClaude        bool
	uninstallDryRun        bool
	uninstallRestoreBackup bool
	uninstallKeepDaemon    bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install SideGuard hooks and MCP wraps for Cursor and Claude Code",
	Long:  "Discovers client configs, backs them up, wraps MCP servers with sideguard wrap, merges hooks, installs the daemon LaunchAgent, and on macOS registers the menu-bar tray LaunchAgent (use --headless to skip tray).",
	RunE: func(_ *cobra.Command, _ []string) error {
		_, err := install.Run(install.Options{
			Cursor:     installCursor,
			Claude:     installClaude,
			DryRun:     installDryRun,
			Discover:   installDiscover,
			SkipDaemon: installSkipDaemon,
			Headless:   installHeadless,
			Dev:        installDev,
		})
		return err
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove SideGuard hooks, MCP wraps, and daemon LaunchAgent",
	Long:  "Default: surgically removes SideGuard hook entries and unwraps MCP servers in-place (preserves your hooks), and removes daemon and tray LaunchAgents on macOS. Use --restore-backup to restore the oldest pre-install backup per file instead.",
	RunE: func(_ *cobra.Command, _ []string) error {
		_, err := install.Uninstall(install.Options{
			Cursor:        uninstallCursor,
			Claude:        uninstallClaude,
			DryRun:        uninstallDryRun,
			RestoreBackup: uninstallRestoreBackup,
			KeepDaemon:    uninstallKeepDaemon,
		})
		return err
	},
}

func init() {
	installCmd.Flags().BoolVar(&installCursor, "cursor", false, "install for Cursor only")
	installCmd.Flags().BoolVar(&installClaude, "claude", false, "install for Claude Code only")
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "print planned changes without writing files")
	installCmd.Flags().BoolVar(&installDiscover, "discover", false, "list discovered MCP servers and config paths")
	installCmd.Flags().BoolVar(&installSkipDaemon, "skip-daemon", false, "skip daemon LaunchAgent install (hooks/MCP only)")
	installCmd.Flags().BoolVar(&installHeadless, "headless", false, "skip menu-bar tray LaunchAgent (SSH/CI installs; macOS only)")
	installCmd.Flags().BoolVar(&installDev, "dev", false, "write repo-scoped workspace dev policy (.sideguard/policy.yaml) for make/go/scripts")

	uninstallCmd.Flags().BoolVar(&uninstallCursor, "cursor", false, "uninstall Cursor configs only")
	uninstallCmd.Flags().BoolVar(&uninstallClaude, "claude", false, "uninstall Claude Code configs only")
	uninstallCmd.Flags().BoolVar(&uninstallDryRun, "dry-run", false, "print planned changes without writing files")
	uninstallCmd.Flags().BoolVar(&uninstallRestoreBackup, "restore-backup", false, "restore oldest backup per file instead of surgical removal")
	uninstallCmd.Flags().BoolVar(&uninstallKeepDaemon, "keep-daemon", false, "leave the daemon and tray LaunchAgents installed")

	rootCmd.AddCommand(installCmd, uninstallCmd)
}
