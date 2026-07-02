package daemon

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

// launchctlDomain returns the user GUI domain (e.g. gui/501).
func launchctlDomain(uid string) string {
	return fmt.Sprintf("gui/%s", uid)
}

// launchctlServiceID returns the fully qualified service id (e.g. gui/501/com.sideguard.daemon).
func launchctlServiceID(domain, label string) string {
	return fmt.Sprintf("%s/%s", domain, label)
}

// isBootoutNotLoaded reports whether launchctl bootout failed because the service was not loaded.
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

// isBootstrapAlreadyLoaded reports whether launchctl bootstrap failed because the agent is already loaded.
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
		// Fall back to bootout by plist path (works when label lookup differs).
		if out2, err2 := runner.run("bootout", domain, plistPath); err2 != nil && !isBootoutNotLoaded(out2, err2) {
			// Best-effort unload before reinstall; ignore residual errors.
			_ = out
			_ = out2
		}
	}
}

func bootstrapLaunchAgent(runner launchctlRunner, domain, plistPath string) (string, error) {
	out, err := runner.run("bootstrap", domain, plistPath)
	return out, err
}

func kickstartLaunchAgent(runner launchctlRunner, domain, label string) (string, error) {
	serviceID := launchctlServiceID(domain, label)
	out, err := runner.run("kickstart", "-k", serviceID)
	return out, err
}

func formatLaunchctlInstallError(domain, label string, bootstrapOut string, bootstrapErr error, kickstartOut string, kickstartErr error) error {
	uid := strings.TrimPrefix(domain, "gui/")
	manual := fmt.Sprintf("launchctl bootout gui/%s %s", uid, label)
	hint := "the LaunchAgent may already be loaded from a prior install or `sideguard daemon start`"

	var b strings.Builder
	fmt.Fprintf(&b, "launchctl bootstrap failed: %v", bootstrapErr)
	if bootstrapOut != "" {
		fmt.Fprintf(&b, ": %s", strings.TrimSpace(bootstrapOut))
	}
	fmt.Fprintf(&b, " (%s). ", hint)
	fmt.Fprintf(&b, "Try unloading manually: %s, then run install again.", manual)
	if kickstartErr != nil {
		fmt.Fprintf(&b, " kickstart fallback also failed: %v", kickstartErr)
		if kickstartOut != "" {
			fmt.Fprintf(&b, ": %s", strings.TrimSpace(kickstartOut))
		}
	}
	return fmt.Errorf("%s", b.String())
}

func loadLaunchAgent(runner launchctlRunner, uid, plistPath string) error {
	domain := launchctlDomain(uid)
	label := paths.LaunchAgentLabel

	bootoutLaunchAgent(runner, domain, label, plistPath)

	bootstrapOut, bootstrapErr := bootstrapLaunchAgent(runner, domain, plistPath)
	if bootstrapErr == nil {
		return nil
	}

	if isBootstrapAlreadyLoaded(bootstrapOut, bootstrapErr) {
		kickstartOut, kickstartErr := kickstartLaunchAgent(runner, domain, label)
		if kickstartErr == nil {
			return nil
		}
		return formatLaunchctlInstallError(domain, label, bootstrapOut, bootstrapErr, kickstartOut, kickstartErr)
	}

	return formatLaunchctlInstallError(domain, label, bootstrapOut, bootstrapErr, "", nil)
}

// InstallService writes the LaunchAgent plist and loads it with launchctl.
// Uses LaunchAgent (user session), not LaunchDaemon. See vgf-phase-2.0-daemon-core.md.
// Idempotent: unloads any existing agent before bootstrap; kickstarts if already loaded.
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
	logPath := filepath.Join(home, paths.RunSubdir, "daemon.log")

	plistPath, err := paths.LaunchAgentPath()
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
		<string>daemon</string>
		<string>run</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, paths.LaunchAgentLabel, exe, logPath, logPath)

	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	uid := fmt.Sprintf("%d", os.Getuid())
	return loadLaunchAgent(runner, uid, plistPath)
}

// LaunchAgentPlistPath returns the installed plist path for handoff documentation.
func LaunchAgentPlistPath() (string, error) {
	return paths.LaunchAgentPath()
}

// UninstallService unloads the LaunchAgent and removes its plist.
// Idempotent: missing plist or unloaded service is not an error.
// See docs/plans/2026-07-01-1418-uninstall-architecture/ (uia-phase-2.0-daemon-lifecycle.md).
func UninstallService() error {
	return uninstallServiceWithRunner(execLaunchctlRunner{})
}

func uninstallServiceWithRunner(runner launchctlRunner) error {
	plistPath, err := paths.LaunchAgentPath()
	if err != nil {
		return err
	}

	uid := fmt.Sprintf("%d", os.Getuid())
	domain := launchctlDomain(uid)
	bootoutLaunchAgent(runner, domain, paths.LaunchAgentLabel, plistPath)

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove LaunchAgent plist: %w", err)
	}
	return nil
}
