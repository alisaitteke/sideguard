// Package detect is VibeGuard's local decision engine. It evaluates the
// execution-free shell.IR produced by internal/shell against embedded YAML
// category rules (plus optional user rules in ~/.vibeguard/rules) and returns a
// Result carrying an allow/deny/ask action, matched rule ids, a numeric risk
// score, and the categories that fired. It performs pure, offline pattern
// matching — no network access and no command execution.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-2.0-detect.md).
package detect

// Category names the class of risk a rule detects. Each category maps to a
// default auto-mode action and a default severity used by scoring.
type Category string

const (
	// CategoryDestructive is irreversible data/disk destruction (rm -rf, dd, mkfs).
	CategoryDestructive Category = "destructive"
	// CategoryExfil is data exfiltration (piping downloads to a shell, POSTing files).
	CategoryExfil Category = "exfil"
	// CategoryReverseShell is a callback/reverse-shell primitive (nc -e, /dev/tcp).
	CategoryReverseShell Category = "reverse_shell"
	// CategoryPrivesc is privilege escalation (setuid, sudoers, chown root).
	CategoryPrivesc Category = "privesc"
	// CategoryBypass is tampering with VibeGuard's own config/hooks/daemon. It is
	// non-overridable: bypass rules may only originate from embedded packs.
	CategoryBypass Category = "bypass"
	// CategoryCredentialAccess is reading secrets (/etc/shadow, ssh keys, .env).
	CategoryCredentialAccess Category = "credential_access"
	// CategoryPersistence is establishing persistence (cron, launchd, rc files).
	CategoryPersistence Category = "persistence"
	// CategoryNetworkMutation is mutating host networking (iptables, pfctl, routes).
	CategoryNetworkMutation Category = "network_mutation"
	// CategoryInterpreterEscape is an interpreter one-liner hiding a nested command
	// (bash -c, python -c); its NestedCommands are re-evaluated.
	CategoryInterpreterEscape Category = "interpreter_escape"
	// CategoryObfuscationCarrier is an encoding/decoding carrier (base64 -d, xxd -r);
	// it only boosts the risk score, never decides on its own.
	CategoryObfuscationCarrier Category = "obfuscation_carrier"
)

// Severity ranks a single rule match for scoring aggregation.
type Severity string

const (
	// SeverityCritical forces deny on its own (critical categories).
	SeverityCritical Severity = "critical"
	// SeverityHigh is a serious match; two of them deny, one asks.
	SeverityHigh Severity = "high"
	// SeverityMedium is a moderate match; two of them ask.
	SeverityMedium Severity = "medium"
	// SeverityLow is a weak signal (e.g. obfuscation carriers).
	SeverityLow Severity = "low"
)

// categoryMeta records the default severity and criticality of a category.
type categoryMeta struct {
	defaultSeverity Severity
	critical        bool
}

// categoryTable is the single source of truth for known categories, their
// default severity (used when a rule omits severity), and whether a match in
// the category is by itself sufficient to deny in auto mode.
var categoryTable = map[Category]categoryMeta{
	CategoryDestructive:        {SeverityCritical, true},
	CategoryExfil:              {SeverityCritical, true},
	CategoryReverseShell:       {SeverityCritical, true},
	CategoryPrivesc:            {SeverityCritical, true},
	CategoryBypass:             {SeverityCritical, true},
	CategoryCredentialAccess:   {SeverityHigh, false},
	CategoryPersistence:        {SeverityHigh, false},
	CategoryNetworkMutation:    {SeverityMedium, false},
	CategoryInterpreterEscape:  {SeverityMedium, false},
	CategoryObfuscationCarrier: {SeverityLow, false},
}

// knownCategory reports whether c is a recognized category.
func knownCategory(c Category) bool {
	_, ok := categoryTable[c]
	return ok
}

// isCriticalCategory reports whether a single match in c denies in auto mode.
func isCriticalCategory(c Category) bool {
	m, ok := categoryTable[c]
	return ok && m.critical
}

// defaultSeverity returns the severity a rule inherits when it omits one.
func defaultSeverity(c Category) Severity {
	if m, ok := categoryTable[c]; ok {
		return m.defaultSeverity
	}
	return SeverityMedium
}

// safeArgv0 is the allow-list of common, low-risk developer command names.
// When no rule matches AND the command's argv0 is in this set, the engine
// returns allow (so everyday tooling passes silently). Interpreters that carry
// arbitrary payloads (bash, sh, python, …) are deliberately excluded.
var safeArgv0 = map[string]struct{}{
	"git": {}, "go": {}, "make": {}, "gofmt": {}, "cargo": {}, "rustc": {},
	"npm": {}, "npx": {}, "yarn": {}, "pnpm": {}, "node": {}, "deno": {}, "bun": {},
	"pip": {}, "pip3": {}, "poetry": {}, "uv": {}, "tsc": {}, "jest": {}, "pytest": {},
	"docker": {}, "kubectl": {}, "helm": {}, "terraform": {}, "brew": {},
	"ls": {}, "cat": {}, "echo": {}, "pwd": {}, "cd": {}, "head": {}, "tail": {},
	"grep": {}, "rg": {}, "find": {}, "fd": {}, "wc": {}, "sort": {}, "uniq": {},
	"diff": {}, "which": {}, "cp": {}, "mv": {}, "mkdir": {}, "touch": {},
}

// isSafeArgv0 reports whether argv0 is a recognized low-risk command name.
func isSafeArgv0(argv0 string) bool {
	if argv0 == "" {
		return false
	}
	_, ok := safeArgv0[argv0]
	return ok
}
