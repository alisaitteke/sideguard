package detect

import (
	"testing"

	"github.com/alisaitteke/sideguard/internal/policy"
)

func TestDecideNoMatchSafeArgv0Allows(t *testing.T) {
	action, score, _ := decide(nil, "git")
	if action != policy.ActionAllow {
		t.Fatalf("safe argv0 with no match: got %q, want allow", action)
	}
	if score != 0 {
		t.Fatalf("expected zero score, got %d", score)
	}
}

func TestDecideNoMatchUnknownArgv0Asks(t *testing.T) {
	action, _, _ := decide(nil, "somethingweird")
	if action != policy.ActionAsk {
		t.Fatalf("unknown argv0 with no match: got %q, want ask", action)
	}
}

func TestDecideBypassAlwaysDeny(t *testing.T) {
	// Bypass is critical, but assert it denies even if flagged low severity.
	matches := []ruleMatch{{id: "b", category: CategoryBypass, severity: SeverityLow}}
	action, _, _ := decide(matches, "sideguard")
	if action != policy.ActionDeny {
		t.Fatalf("bypass match: got %q, want deny", action)
	}
}

func TestDecideCriticalCategoryDenies(t *testing.T) {
	matches := []ruleMatch{{id: "d", category: CategoryDestructive, severity: SeverityCritical}}
	action, _, _ := decide(matches, "rm")
	if action != policy.ActionDeny {
		t.Fatalf("critical category: got %q, want deny", action)
	}
}

func TestDecideTwoHighDenies(t *testing.T) {
	matches := []ruleMatch{
		{id: "h1", category: CategoryCredentialAccess, severity: SeverityHigh},
		{id: "h2", category: CategoryPersistence, severity: SeverityHigh},
	}
	action, _, _ := decide(matches, "sh")
	if action != policy.ActionDeny {
		t.Fatalf("two high: got %q, want deny", action)
	}
}

func TestDecideOneHighAsks(t *testing.T) {
	matches := []ruleMatch{{id: "h", category: CategoryCredentialAccess, severity: SeverityHigh}}
	action, _, _ := decide(matches, "cat")
	if action != policy.ActionAsk {
		t.Fatalf("one high: got %q, want ask", action)
	}
}

func TestDecideTwoMediumAsks(t *testing.T) {
	matches := []ruleMatch{
		{id: "m1", category: CategoryNetworkMutation, severity: SeverityMedium},
		{id: "m2", category: CategoryInterpreterEscape, severity: SeverityMedium},
	}
	action, _, _ := decide(matches, "sh")
	if action != policy.ActionAsk {
		t.Fatalf("two medium: got %q, want ask", action)
	}
}

func TestDecideScoreAccumulates(t *testing.T) {
	matches := []ruleMatch{
		{id: "c", category: CategoryDestructive, severity: SeverityCritical},
		{id: "l", category: CategoryObfuscationCarrier, severity: SeverityLow},
	}
	_, score, _ := decide(matches, "rm")
	if score != severityPoints(SeverityCritical)+severityPoints(SeverityLow) {
		t.Fatalf("unexpected score %d", score)
	}
}

func TestIsSafeArgv0(t *testing.T) {
	for _, ok := range []string{"git", "go", "make", "docker", "npm"} {
		if !isSafeArgv0(ok) {
			t.Errorf("expected %q safe", ok)
		}
	}
	for _, bad := range []string{"", "rm", "bash", "curl", "chmod"} {
		if isSafeArgv0(bad) {
			t.Errorf("expected %q not safe", bad)
		}
	}
}
