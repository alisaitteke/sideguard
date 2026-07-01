package approvalmode

import "testing"

func TestModeValid(t *testing.T) {
	t.Parallel()
	for _, m := range []Mode{Ask, AutoAllow, AutoDeny} {
		if !m.Valid() {
			t.Fatalf("%q should be valid", m)
		}
	}
	if Mode("bogus").Valid() {
		t.Fatal("bogus should be invalid")
	}
}

func TestParse(t *testing.T) {
	t.Parallel()
	m, err := Parse("auto_allow")
	if err != nil || m != AutoAllow {
		t.Fatalf("Parse: got %q err=%v", m, err)
	}
	if _, err := Parse("nope"); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestDecisionAndReason(t *testing.T) {
	t.Parallel()
	if Ask.Decision() != "" || Ask.AutoReason() != "" {
		t.Fatal("ask should not auto-decide")
	}
	if AutoAllow.Decision() != "allow" || AutoAllow.AutoReason() != "auto-approved by mode" {
		t.Fatal("auto_allow mismatch")
	}
	if AutoDeny.Decision() != "deny" || AutoDeny.AutoReason() != "auto-denied by mode" {
		t.Fatal("auto_deny mismatch")
	}
}

func TestParseCLI(t *testing.T) {
	t.Parallel()
	m, err := ParseCLI("auto-allow")
	if err != nil || m != AutoAllow {
		t.Fatalf("ParseCLI: got %q err=%v", m, err)
	}
	m, err = ParseCLI("auto_deny")
	if err != nil || m != AutoDeny {
		t.Fatalf("ParseCLI deny: got %q err=%v", m, err)
	}
}

func TestLabel(t *testing.T) {
	t.Parallel()
	if Ask.Label() != "Ask" || AutoAllow.Label() != "Auto-allow" || AutoDeny.Label() != "Auto-deny" {
		t.Fatal("unexpected labels")
	}
}
