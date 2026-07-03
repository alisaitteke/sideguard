// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Package paths provides XDG-style path helpers for SideGuard local state.
// See docs/plans/2026-07-01-0127-sideguard-foundation/ (vgf-phase-1.0-project-init.md).
package paths

import (
	"os"
	"path/filepath"
)

const (
	// DirName is the base directory under the user's home for all SideGuard state.
	DirName = ".sideguard"

	// RunSubdir holds runtime artifacts such as the Unix socket.
	RunSubdir = "run"

	// SocketFile is the Unix domain socket filename.
	SocketFile = "sideguard.sock"

	// AuditDBFile is the SQLite audit log database filename.
	AuditDBFile = "audit.db"

	// PIDFile is the daemon process id file name.
	PIDFile = "sideguard.pid"

	// LaunchAgentLabel is the launchd label for the user-session daemon.
	LaunchAgentLabel = "com.sideguard.daemon"

	// LaunchAgentFile is the plist filename under ~/Library/LaunchAgents.
	LaunchAgentFile = "com.sideguard.daemon.plist"

	// TrayLaunchAgentLabel is the launchd label for the menu-bar tray.
	TrayLaunchAgentLabel = "com.sideguard.tray"

	// TrayLaunchAgentFile is the tray plist filename under ~/Library/LaunchAgents.
	TrayLaunchAgentFile = "com.sideguard.tray.plist"

	// BackupsSubdir holds timestamped config backups from install.
	BackupsSubdir = "backups"

	// PolicyFile is the YAML policy rules filename.
	PolicyFile = "policy.yaml"

	// ConfigFile is the global LLM and daemon settings filename.
	ConfigFile = "config.yaml"

	// CredentialsFile holds provider API keys (mode 0600).
	CredentialsFile = "credentials.yaml"

	// SignaturesSubdir holds LLM classification prompt files.
	SignaturesSubdir = "signatures"

	// RulesSubdir holds user-supplied detect rule packs (~/.sideguard/rules).
	// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-2.0-detect.md).
	RulesSubdir = "rules"

	// UpdateStateFile persists background update check results.
	// See docs/plans/2026-07-02-1102-github-update/ (vgu-phase-2.0-update-core.md).
	UpdateStateFile = "update-state.json"

	// UpdateSubdir holds downloaded release artifacts under ~/.sideguard/run/update/.
	UpdateSubdir = "update"
)

// Home returns the SideGuard base directory (~/.sideguard).
func Home() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirName), nil
}

// RunDir returns the runtime directory (~/.sideguard/run).
func RunDir() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, RunSubdir), nil
}

// SocketPath returns the Unix socket path (~/.sideguard/run/sideguard.sock).
func SocketPath() (string, error) {
	dir, err := RunDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, SocketFile), nil
}

// AuditDBPath returns the SQLite audit database path (~/.sideguard/audit.db).
func AuditDBPath() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AuditDBFile), nil
}

// PIDPath returns the daemon pid file path (~/.sideguard/run/sideguard.pid).
func PIDPath() (string, error) {
	dir, err := RunDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, PIDFile), nil
}

// BackupsDir returns the install backup directory (~/.sideguard/backups).
func BackupsDir() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, BackupsSubdir), nil
}

// PolicyPath returns the global policy file path (~/.sideguard/policy.yaml).
func PolicyPath() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, PolicyFile), nil
}

// ConfigPath returns the global config file path (~/.sideguard/config.yaml).
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-1.0-contracts.md).
func ConfigPath() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, ConfigFile), nil
}

// CredentialsPath returns the credentials file path (~/.sideguard/credentials.yaml).
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-1.0-contracts.md).
func CredentialsPath() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, CredentialsFile), nil
}

// SignaturesDir returns the LLM signature prompts directory (~/.sideguard/signatures).
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-1.0-contracts.md).
func SignaturesDir() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, SignaturesSubdir), nil
}

// RulesDir returns the user detect-rules directory (~/.sideguard/rules), where
// the detect engine loads optional user rule packs merged after embedded rules.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-2.0-detect.md).
func RulesDir() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, RulesSubdir), nil
}

// LaunchAgentPath returns the LaunchAgent plist path
// (~/Library/LaunchAgents/com.sideguard.daemon.plist).
func LaunchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", LaunchAgentFile), nil
}

// TrayLaunchAgentPath returns the tray LaunchAgent plist path
// (~/Library/LaunchAgents/com.sideguard.tray.plist).
func TrayLaunchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", TrayLaunchAgentFile), nil
}

// UpdateStatePath returns the update checker state file (~/.sideguard/update-state.json).
// See docs/plans/2026-07-02-1102-github-update/ (vgu-phase-2.0-update-core.md).
func UpdateStatePath() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, UpdateStateFile), nil
}

// UpdateRunDir returns the per-version download staging directory
// (~/.sideguard/run/update/<version>/).
func UpdateRunDir(version string) (string, error) {
	dir, err := RunDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, UpdateSubdir, version), nil
}
