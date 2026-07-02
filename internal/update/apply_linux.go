//go:build linux

package update

import (
	"context"
	"log"
	"os"
	"os/exec"

	"github.com/alisaitteke/sideguard/internal/daemon"
	"github.com/alisaitteke/sideguard/internal/paths"
)

const systemdDaemonUnit = "sideguard-daemon.service"

type linuxApplier struct{}

func newPlatformApplier() PlatformApplier {
	return linuxApplier{}
}

// Stop stops the user systemd unit when present, then stops the daemon process (best-effort).
func (linuxApplier) Stop(ctx context.Context) error {
	if hasSystemdDaemonUnit() {
		if err := runSystemctlUser(ctx, "stop", systemdDaemonUnit); err != nil {
			log.Printf("update: systemctl stop: %v", err)
		}
	}
	if err := daemon.StopBestEffort(); err != nil {
		log.Printf("update: stop daemon: %v", err)
	}
	return nil
}

// SwapBinary atomically replaces the running binary on disk.
func (linuxApplier) SwapBinary(ctx context.Context, stagingPath, targetPath string) error {
	_ = ctx
	return atomicSwapBinary(stagingPath, targetPath)
}

// Start starts the user systemd unit when present, otherwise forks the daemon process.
func (linuxApplier) Start(ctx context.Context) error {
	if hasSystemdDaemonUnit() {
		if err := runSystemctlUser(ctx, "start", systemdDaemonUnit); err != nil {
			return err
		}
		return nil
	}
	return daemon.Start("")
}

func hasSystemdDaemonUnit() bool {
	path, err := paths.SystemdUnitPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func runSystemctlUser(ctx context.Context, args ...string) error {
	cmdArgs := append([]string{"--user"}, args...)
	cmd := exec.CommandContext(ctx, "systemctl", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &systemctlError{args: args, out: string(out), err: err}
	}
	return nil
}

type systemctlError struct {
	args []string
	out  string
	err  error
}

func (e *systemctlError) Error() string {
	return e.err.Error() + ": " + e.out
}

func (e *systemctlError) Unwrap() error {
	return e.err
}
