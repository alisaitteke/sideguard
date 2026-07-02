//go:build linux

package paths

import (
	"os"
	"path/filepath"
)

const (
	// SystemdDaemonUnit is the user-level systemd unit filename for the daemon.
	SystemdDaemonUnit = "sideguard-daemon.service"
)

// SystemdUnitPath returns the user systemd unit file path
// (~/.config/systemd/user/sideguard-daemon.service).
func SystemdUnitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user", SystemdDaemonUnit), nil
}
