//go:build darwin

package update

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/daemon"
	"github.com/alisaitteke/vibeguard/internal/paths"
)

type darwinApplier struct{}

func newPlatformApplier() PlatformApplier {
	return darwinApplier{}
}

// Stop stops the daemon (PID) and unloads the tray LaunchAgent (best-effort).
func (darwinApplier) Stop(ctx context.Context) error {
	_ = ctx
	if err := daemon.StopBestEffort(); err != nil {
		log.Printf("update: stop daemon: %v", err)
	}
	bootoutTrayLaunchAgent()
	return nil
}

// SwapBinary atomically replaces the running binary on disk.
func (darwinApplier) SwapBinary(ctx context.Context, stagingPath, targetPath string) error {
	_ = ctx
	return atomicSwapBinary(stagingPath, targetPath)
}

// Start restarts the daemon and kickstarts the tray LaunchAgent when installed.
func (darwinApplier) Start(ctx context.Context) error {
	_ = ctx
	if err := daemon.Start(""); err != nil {
		return err
	}
	if err := kickstartTrayLaunchAgent(); err != nil {
		log.Printf("update: kickstart tray: %v", err)
	}
	return nil
}

type darwinLaunchctlRunner interface {
	run(args ...string) (combinedOutput string, err error)
}

type execDarwinLaunchctl struct{}

func (execDarwinLaunchctl) run(args ...string) (string, error) {
	cmd := exec.Command("launchctl", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func darwinLaunchctlDomain(uid string) string {
	return fmt.Sprintf("gui/%s", uid)
}

func darwinLaunchctlServiceID(domain, label string) string {
	return fmt.Sprintf("%s/%s", domain, label)
}

func isDarwinBootoutNotLoaded(output string, err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(output)
	return strings.Contains(lower, "no such process") ||
		strings.Contains(lower, "could not find") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "service is not running")
}

func bootoutTrayLaunchAgent() {
	plistPath, err := paths.TrayLaunchAgentPath()
	if err != nil {
		return
	}
	runner := execDarwinLaunchctl{}
	uid := fmt.Sprintf("%d", os.Getuid())
	domain := darwinLaunchctlDomain(uid)
	label := paths.TrayLaunchAgentLabel
	serviceID := darwinLaunchctlServiceID(domain, label)
	if out, err := runner.run("bootout", serviceID); err != nil && !isDarwinBootoutNotLoaded(out, err) {
		if out2, err2 := runner.run("bootout", domain, plistPath); err2 != nil && !isDarwinBootoutNotLoaded(out2, err2) {
			log.Printf("update: bootout tray: %v", err2)
		}
	}
}

func kickstartTrayLaunchAgent() error {
	plistPath, err := paths.TrayLaunchAgentPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(plistPath); err != nil {
		return nil
	}
	runner := execDarwinLaunchctl{}
	uid := fmt.Sprintf("%d", os.Getuid())
	domain := darwinLaunchctlDomain(uid)
	serviceID := darwinLaunchctlServiceID(domain, paths.TrayLaunchAgentLabel)
	if _, err := runner.run("kickstart", "-k", serviceID); err != nil {
		return fmt.Errorf("kickstart tray LaunchAgent: %w", err)
	}
	return nil
}
