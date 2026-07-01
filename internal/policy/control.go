package policy

import (
	"regexp"
	"strings"
)

// ControlPlaneCommandPattern matches VibeGuard CLI commands that must auto-allow
// so agents can run approve/deny without deadlocking on their own hooks.
// See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-11.0-checklist.md).
const ControlPlaneCommandPattern = `^vibeguard (pending|approve|deny|ui|status|daemon|uninstall|doctor|policy|clients)(\s|$)`

var controlPlaneCommandRe = regexp.MustCompile(ControlPlaneCommandPattern)

// IsControlPlaneCommand reports whether command is a VibeGuard control-plane CLI invocation.
func IsControlPlaneCommand(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	return controlPlaneCommandRe.MatchString(command)
}
