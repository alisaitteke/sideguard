// Package paths provides XDG-style path helpers for VibeGuard local state.
// See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-1.0-project-init.md).
package paths

import (
	"os"
	"path/filepath"
)

const (
	// DirName is the base directory under the user's home for all VibeGuard state.
	DirName = ".vibeguard"

	// RunSubdir holds runtime artifacts such as the Unix socket.
	RunSubdir = "run"

	// SocketFile is the Unix domain socket filename.
	SocketFile = "vibeguard.sock"

	// AuditDBFile is the SQLite audit log database filename.
	AuditDBFile = "audit.db"

	// PIDFile is the daemon process id file name.
	PIDFile = "vibeguard.pid"

	// LaunchAgentLabel is the launchd label for the user-session daemon.
	LaunchAgentLabel = "com.vibeguard.daemon"

	// LaunchAgentFile is the plist filename under ~/Library/LaunchAgents.
	LaunchAgentFile = "com.vibeguard.daemon.plist"

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
)

// Home returns the VibeGuard base directory (~/.vibeguard).
func Home() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirName), nil
}

// RunDir returns the runtime directory (~/.vibeguard/run).
func RunDir() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, RunSubdir), nil
}

// SocketPath returns the Unix socket path (~/.vibeguard/run/vibeguard.sock).
func SocketPath() (string, error) {
	dir, err := RunDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, SocketFile), nil
}

// AuditDBPath returns the SQLite audit database path (~/.vibeguard/audit.db).
func AuditDBPath() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AuditDBFile), nil
}

// PIDPath returns the daemon pid file path (~/.vibeguard/run/vibeguard.pid).
func PIDPath() (string, error) {
	dir, err := RunDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, PIDFile), nil
}

// BackupsDir returns the install backup directory (~/.vibeguard/backups).
func BackupsDir() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, BackupsSubdir), nil
}

// PolicyPath returns the global policy file path (~/.vibeguard/policy.yaml).
func PolicyPath() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, PolicyFile), nil
}

// ConfigPath returns the global config file path (~/.vibeguard/config.yaml).
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-1.0-contracts.md).
func ConfigPath() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, ConfigFile), nil
}

// CredentialsPath returns the credentials file path (~/.vibeguard/credentials.yaml).
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-1.0-contracts.md).
func CredentialsPath() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, CredentialsFile), nil
}

// SignaturesDir returns the LLM signature prompts directory (~/.vibeguard/signatures).
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-1.0-contracts.md).
func SignaturesDir() (string, error) {
	base, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, SignaturesSubdir), nil
}

// LaunchAgentPath returns the LaunchAgent plist path
// (~/Library/LaunchAgents/com.vibeguard.daemon.plist).
func LaunchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", LaunchAgentFile), nil
}
