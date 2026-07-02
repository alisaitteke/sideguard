package tray

import (
	"fmt"
	"os"
	"os/exec"
)

// SpawnDetachedUpdateApply starts `sideguard update apply --restart --yes` detached from the tray.
// The tray process should exit immediately after a successful spawn so the binary can be replaced.
// See docs/plans/2026-07-02-1102-github-update/ (vgu-phase-4.0-tray-update.md).
func SpawnDetachedUpdateApply() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	cmd := exec.Command(exe, "update", "apply", "--restart", "--yes")
	cmd.Stdout = nil
	cmd.Stderr = nil
	setDetached(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start update apply: %w", err)
	}
	return nil
}

// HandleInstallUpdate guards double-clicks, spawns detached apply, and invokes quit on success.
func HandleInstallUpdate(state *UpdateState, quit func()) {
	if state == nil || quit == nil {
		return
	}
	if !state.TryBeginInstall() {
		return
	}
	if err := SpawnDetachedUpdateApply(); err != nil {
		state.SetInstalling(false)
		return
	}
	quit()
}
