// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Package cmd defines the SideGuard cobra CLI surface.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/daemon"
)

// Version is set at build time via -ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "sideguard",
	Short: "Local security layer for AI coding agents",
	Long:  "SideGuard intercepts shell commands and MCP tool calls, holding them for user approval.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return cmd.Help()
		}
		return runRootDefault()
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(pendingCmd)
	rootCmd.AddCommand(approveCmd)
	rootCmd.AddCommand(denyCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("sideguard %s\n", Version)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon health and pending approval count",
	RunE: func(_ *cobra.Command, _ []string) error {
		line, err := daemon.Status()
		if err != nil {
			return daemonNotRunningError("status")
		}
		fmt.Println(line)
		return nil
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the SideGuard background daemon",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon in the background",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := daemon.Start(Version); err != nil {
			return err
		}
		fmt.Println("daemon started")
		return nil
	},
}

var daemonRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the daemon in the foreground (LaunchAgent entrypoint)",
	Hidden: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		return daemon.Run(Version)
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := daemon.Stop(); err != nil {
			return err
		}
		fmt.Println("daemon stopped")
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report daemon health",
	RunE: func(_ *cobra.Command, _ []string) error {
		line, err := daemon.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "daemon is not running: %v\n", err)
			fmt.Fprintf(os.Stderr, "hint: start the daemon with `sideguard daemon start`\n")
			fmt.Fprintf(os.Stderr, "health endpoint: %s\n", api.HealthURL())
			return err
		}
		fmt.Println(line)
		return nil
	},
}

var daemonInstallServiceCmd = &cobra.Command{
	Use:   "install-service",
	Short: "Install LaunchAgent plist and load via launchctl",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := daemon.InstallService(); err != nil {
			return err
		}
		path, _ := daemon.LaunchAgentPlistPath()
		fmt.Printf("LaunchAgent installed: %s\n", path)
		return nil
	},
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd, daemonRunCmd, daemonStopCmd, daemonStatusCmd, daemonInstallServiceCmd)
}

func daemonNotRunningError(command string) error {
	err := fmt.Errorf("daemon is not running")
	fmt.Fprintf(os.Stderr, "%s: %v\n", command, err)
	fmt.Fprintf(os.Stderr, "hint: start the daemon with `sideguard daemon start`\n")
	fmt.Fprintf(os.Stderr, "health endpoint: %s\n", api.HealthURL())
	return err
}
