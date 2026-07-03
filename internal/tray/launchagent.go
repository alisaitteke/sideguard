// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package tray

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alisaitteke/sideguard/internal/paths"
)

// launchctlRunner executes launchctl subcommands. Tests inject a mock implementation.
type launchctlRunner interface {
	run(args ...string) (combinedOutput string, err error)
}

type execLaunchctlRunner struct{}

func (execLaunchctlRunner) run(args ...string) (string, error) {
	cmd := exec.Command("launchctl", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func launchctlDomain(uid string) string {
	return fmt.Sprintf("gui/%s", uid)
}

func launchctlServiceID(domain, label string) string {
	return fmt.Sprintf("%s/%s", domain, label)
}

func isBootoutNotLoaded(output string, err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(output)
	return strings.Contains(lower, "no such process") ||
		strings.Contains(lower, "could not find") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "service is not running")
}

func isBootstrapAlreadyLoaded(output string, err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(output)
	return strings.Contains(lower, "bootstrap failed: 5") ||
		strings.Contains(lower, "input/output error") ||
		strings.Contains(lower, "already loaded") ||
		strings.Contains(lower, "service already bootstrapped")
}

func bootoutLaunchAgent(runner launchctlRunner, domain, label, plistPath string) {
	serviceID := launchctlServiceID(domain, label)
	if out, err := runner.run("bootout", serviceID); err != nil && !isBootoutNotLoaded(out, err) {
		if out2, err2 := runner.run("bootout", domain, plistPath); err2 != nil && !isBootoutNotLoaded(out2, err2) {
			_ = out
			_ = out2
		}
	}
}

func loadTrayLaunchAgent(runner launchctlRunner, uid, plistPath string) error {
	domain := launchctlDomain(uid)
	label := paths.TrayLaunchAgentLabel

	bootoutLaunchAgent(runner, domain, label, plistPath)

	bootstrapOut, bootstrapErr := runner.run("bootstrap", domain, plistPath)
	if bootstrapErr == nil {
		return nil
	}

	if isBootstrapAlreadyLoaded(bootstrapOut, bootstrapErr) {
		serviceID := launchctlServiceID(domain, label)
		if _, kickstartErr := runner.run("kickstart", "-k", serviceID); kickstartErr == nil {
			return nil
		}
	}

	manual := fmt.Sprintf("launchctl bootout gui/%s %s", uid, label)
	return fmt.Errorf("launchctl bootstrap failed: %v: %s — try: %s", bootstrapErr, strings.TrimSpace(bootstrapOut), manual)
}

// InstallService writes the tray LaunchAgent plist and loads it with launchctl.
// KeepAlive is false (tray exits when user quits); RunAtLoad starts tray at login.
// See docs/plans/2026-07-01-1355-go-systray-tray/ (gst-phase-4.0-macos-packaging.md).
func InstallService() error {
	return installServiceWithRunner(execLaunchctlRunner{})
}

func installServiceWithRunner(runner launchctlRunner) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("eval symlinks: %w", err)
	}

	home, err := paths.Home()
	if err != nil {
		return err
	}
	logPath := filepath.Join(home, paths.RunSubdir, "tray.log")

	plistPath, err := paths.TrayLaunchAgentPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>tray</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, paths.TrayLaunchAgentLabel, exe, logPath, logPath)

	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	uid := fmt.Sprintf("%d", os.Getuid())
	return loadTrayLaunchAgent(runner, uid, plistPath)
}

// LaunchAgentPlistPath returns the installed tray plist path for documentation.
func LaunchAgentPlistPath() (string, error) {
	return paths.TrayLaunchAgentPath()
}

// UninstallService unloads the tray LaunchAgent and removes its plist.
// Idempotent: missing plist or unloaded service is not an error.
func UninstallService() error {
	return uninstallServiceWithRunner(execLaunchctlRunner{})
}

func uninstallServiceWithRunner(runner launchctlRunner) error {
	plistPath, err := paths.TrayLaunchAgentPath()
	if err != nil {
		return err
	}

	uid := fmt.Sprintf("%d", os.Getuid())
	domain := launchctlDomain(uid)
	bootoutLaunchAgent(runner, domain, paths.TrayLaunchAgentLabel, plistPath)

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove tray LaunchAgent plist: %w", err)
	}
	return nil
}
