package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/sideguard/internal/tray"
)

var trayCmd = &cobra.Command{
	Use:   "tray",
	Short: "Menu-bar tray for pending approvals",
	Long: `Run SideGuard in the menu bar as a background process.

On macOS, clicking the menu-bar icon toggles a native popover panel below the
icon (not a context menu). The panel polls the daemon every ~2s and shows:
  • Daemon health and pending count
  • Approval mode control (Ask / Auto-allow / Auto-deny)
  • Up to 10 pending rows with flat Allow and Deny buttons per row
  • Auto-open when new pending approvals arrive while the panel is hidden
  • Use "sideguard ui" for more than 10 pending items

Tooltip and menu-bar title reflect pending count; when the daemon is stopped the
tray stays visible with an unreachable status.

On non-macOS platforms, the tray uses a systray context menu instead.

Start the daemon first (sideguard daemon start), then run this command.
Use "sideguard mode set auto-allow" or the tray mode control for hands-off local dev.

Requires CGO_ENABLED=1 at build time and an active GUI session (not SSH-only).

On macOS you can also run dist/SideGuard Tray.app (make tray-app) or install a
LaunchAgent: sideguard tray install-service`,
	RunE: runTray,
}

var trayInstallServiceCmd = &cobra.Command{
	Use:   "install-service",
	Short: "Install LaunchAgent plist to start tray at login",
	Long: `Writes ~/Library/LaunchAgents/com.sideguard.tray.plist and loads it via launchctl.

The tray starts at login (RunAtLoad) but does not respawn when quit (KeepAlive false).
Start the daemon separately (sideguard daemon start or daemon install-service).

Uses the current sideguard binary path at install time.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := tray.InstallService(); err != nil {
			return err
		}
		path, _ := tray.LaunchAgentPlistPath()
		fmt.Printf("Tray LaunchAgent installed: %s\n", path)
		return nil
	},
}

func init() {
	trayCmd.AddCommand(trayInstallServiceCmd)
	rootCmd.AddCommand(trayCmd)
}

func runTray(_ *cobra.Command, _ []string) error {
	return tray.Run(tray.Options{Version: Version})
}
