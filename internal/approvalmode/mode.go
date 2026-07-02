// Package approvalmode defines the daemon-persisted global approval mode.
// See docs/plans/2026-07-01-1515-global-approval-mode/ (gam-phase-1.0-store-mode.md).
package approvalmode

import (
	"fmt"
	"strings"
)

// Mode is the daemon-wide approval behavior for policy-ask requests.
type Mode string

const (
	// Ask queues requests for manual Allow/Deny.
	Ask Mode = "ask"
	// Auto applies smart triage: detect allow/deny at hook level; uncertain → LLM or queue.
	Auto Mode = "auto"
	// AutoAllow auto-allows every queued request (audit logged).
	AutoAllow Mode = "auto_allow"
	// AutoDeny auto-denies every queued request (audit logged).
	AutoDeny Mode = "auto_deny"
)

// SettingKey is the daemon_settings row key for approval mode.
const SettingKey = "approval_mode"

// Valid reports whether m is a known mode value.
func (m Mode) Valid() bool {
	switch m {
	case Ask, Auto, AutoAllow, AutoDeny:
		return true
	default:
		return false
	}
}

// Parse converts a stored or API mode string into Mode.
func Parse(s string) (Mode, error) {
	m := Mode(strings.TrimSpace(s))
	if !m.Valid() {
		return "", fmt.Errorf("invalid approval mode %q", s)
	}
	return m, nil
}

// Label returns a short human-readable label for UI surfaces.
func (m Mode) Label() string {
	switch m {
	case Auto:
		return "Auto"
	case AutoAllow:
		return "Auto-allow"
	case AutoDeny:
		return "Auto-deny"
	default:
		return "Ask"
	}
}

// Decision returns the auto decision for this mode: "" for ask, "allow" or "deny" otherwise.
func (m Mode) Decision() string {
	switch m {
	case AutoAllow:
		return "allow"
	case AutoDeny:
		return "deny"
	default:
		return ""
	}
}

// AutoReason returns the audit/user message when auto-deciding in this mode.
func (m Mode) AutoReason() string {
	switch m {
	case AutoAllow:
		return "auto-approved by mode"
	case AutoDeny:
		return "auto-denied by mode"
	default:
		return ""
	}
}

// ParseCLI converts CLI aliases (auto-allow, auto-deny) to store values.
func ParseCLI(s string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ask":
		return Ask, nil
	case "auto":
		return Auto, nil
	case "auto-allow", "auto_allow":
		return AutoAllow, nil
	case "auto-deny", "auto_deny":
		return AutoDeny, nil
	default:
		return "", fmt.Errorf("invalid approval mode %q (use ask, auto, auto-allow, or auto-deny)", s)
	}
}
