package detect

import "github.com/alisaitteke/sideguard/internal/policy"

// ruleMatch is a single rule that fired against an IR, carrying the metadata the
// scorer needs to reach a decision.
type ruleMatch struct {
	id       string
	category Category
	severity Severity
	reason   string
}

// severityPoints assigns a numeric weight to a severity for the aggregate risk
// score. The score is advisory (surfaced for audit/telemetry); the action is
// decided by the tiered rules in decide, not by the raw score.
func severityPoints(s Severity) int {
	switch s {
	case SeverityCritical:
		return 100
	case SeverityHigh:
		return 40
	case SeverityMedium:
		return 15
	case SeverityLow:
		return 5
	default:
		return 0
	}
}

// severityRank orders severities so the scorer can pick the most serious
// matched rule's reason as the human-facing explanation.
func severityRank(s Severity) int {
	switch s {
	case SeverityCritical:
		return 3
	case SeverityHigh:
		return 2
	case SeverityMedium:
		return 1
	default:
		return 0
	}
}

// decide maps a set of rule matches to an action, an aggregate score, and a
// reason, following the Phase 2 scoring contract (deny > ask > allow):
//
//	any bypass match              → deny (non-overridable)
//	any critical category match   → deny
//	≥2 high-severity matches      → deny
//	1 high OR ≥2 medium matches   → ask
//	no match + safe argv0 profile → allow
//	otherwise                     → ask
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-2.0-detect.md).
func decide(matches []ruleMatch, argv0 string) (policy.Action, int, string) {
	if len(matches) == 0 {
		if isSafeArgv0(argv0) {
			return policy.ActionAllow, 0, "no risky pattern matched; recognized safe command profile"
		}
		return policy.ActionAsk, 0, "no rule matched; queued for review"
	}

	var (
		score       int
		highCount   int
		mediumCount int
		hasBypass   bool
		hasCritical bool
		bestRank    = -1
		bestReason  string
	)

	for _, m := range matches {
		score += severityPoints(m.severity)
		if m.category == CategoryBypass {
			hasBypass = true
		}
		if isCriticalCategory(m.category) {
			hasCritical = true
		}
		switch m.severity {
		case SeverityHigh:
			highCount++
		case SeverityMedium:
			mediumCount++
		}
		if r := severityRank(m.severity); r > bestRank && m.reason != "" {
			bestRank = r
			bestReason = m.reason
		}
	}

	switch {
	case hasBypass:
		return policy.ActionDeny, score, reasonOr(bestReason, "blocked: SideGuard control-plane tampering")
	case hasCritical:
		return policy.ActionDeny, score, reasonOr(bestReason, "blocked: critical risk category")
	case highCount >= 2:
		return policy.ActionDeny, score, reasonOr(bestReason, "blocked: multiple high-severity signals")
	case highCount >= 1 || mediumCount >= 2:
		return policy.ActionAsk, score, reasonOr(bestReason, "review required: elevated risk")
	default:
		return policy.ActionAsk, score, reasonOr(bestReason, "review required")
	}
}

// reasonOr returns primary when non-empty, otherwise fallback.
func reasonOr(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
