//go:build linux

// Linux systemd user service install stub.
// See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-8.0-hardening.md).
package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alisaitteke/vibeguard/internal/paths"
)

// SystemdUnitPath returns the user systemd unit file path.
func SystemdUnitPath() (string, error) {
	return paths.SystemdUnitPath()
}

// RenderSystemdUnit returns a user-level systemd unit for the VibeGuard daemon.
func RenderSystemdUnit(exe string) (string, error) {
	home, err := paths.Home()
	if err != nil {
		return "", err
	}
	logPath := filepath.Join(home, paths.RunSubdir, "daemon.log")

	return fmt.Sprintf(`[Unit]
Description=VibeGuard approval daemon
After=network.target

[Service]
Type=simple
ExecStart=%s daemon run
Restart=on-failure
RestartSec=3
StandardOutput=append:%s
StandardError=append:%s

[Install]
WantedBy=default.target
`, exe, logPath, logPath), nil
}

// InstallLinuxService writes the systemd user unit template (stub — does not enable/start).
func InstallLinuxService() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("eval symlinks: %w", err)
	}

	unitPath, err := SystemdUnitPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return fmt.Errorf("create systemd user dir: %w", err)
	}

	content, err := RenderSystemdUnit(exe)
	if err != nil {
		return err
	}
	if err := os.WriteFile(unitPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write systemd unit: %w", err)
	}
	return nil
}
