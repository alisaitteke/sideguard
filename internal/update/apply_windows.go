//go:build windows

package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/alisaitteke/vibeguard/internal/daemon"
	"github.com/alisaitteke/vibeguard/internal/paths"
)

type windowsApplier struct{}

func newPlatformApplier() PlatformApplier {
	return windowsApplier{}
}

// Stop stops the daemon via the PID file (best-effort).
func (windowsApplier) Stop(ctx context.Context) error {
	_ = ctx
	if err := daemon.StopBestEffort(); err != nil {
		return err
	}
	return nil
}

// SwapBinary stages target.new and launches a helper script that swaps after this process exits.
func (windowsApplier) SwapBinary(ctx context.Context, stagingPath, targetPath string) error {
	newPath := targetPath + ".new"
	if err := copyFile(stagingPath, newPath); err != nil {
		return err
	}

	runDir, err := paths.RunDir()
	if err != nil {
		return err
	}
	updateDir := filepath.Join(runDir, paths.UpdateSubdir)
	if err := os.MkdirAll(updateDir, 0o700); err != nil {
		return err
	}

	helperPath := filepath.Join(updateDir, "apply-helper.cmd")
	script := renderWindowsApplyHelper(targetPath, newPath, helperPath)
	if err := os.WriteFile(helperPath, []byte(script), 0o600); err != nil {
		return fmt.Errorf("write apply helper: %w", err)
	}

	cmd := exec.CommandContext(ctx, "cmd.exe", "/C", "start", "/B", "", helperPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start apply helper: %w", err)
	}
	return nil
}

// Start is a no-op on Windows; the deferred helper script restarts the daemon.
func (windowsApplier) Start(context.Context) error {
	return nil
}

func renderWindowsApplyHelper(targetPath, newPath, helperPath string) string {
	return fmt.Sprintf(`@echo off
setlocal EnableExtensions
set "TARGET=%s"
set "NEW=%s"
:retry
ping -n 2 127.0.0.1 >nul
move /Y "%%NEW%%" "%%TARGET%%" >nul 2>&1
if errorlevel 1 goto retry
start "" "%%TARGET%%" daemon run
del "%s"
`, targetPath, newPath, helperPath)
}
