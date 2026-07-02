package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/policy"
	"github.com/alisaitteke/vibeguard/internal/shell"
)

func TestLoadEmbeddedCompiles(t *testing.T) {
	rules, err := loadEmbedded()
	if err != nil {
		t.Fatalf("loadEmbedded: %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("expected embedded rules, got none")
	}

	// Every critical category must have at least one embedded rule.
	want := []Category{
		CategoryDestructive, CategoryExfil, CategoryReverseShell,
		CategoryPrivesc, CategoryBypass,
	}
	present := map[Category]bool{}
	for _, r := range rules {
		present[r.category] = true
	}
	for _, c := range want {
		if !present[c] {
			t.Errorf("no embedded rule for critical category %q", c)
		}
	}
}

func TestUserBypassRuleDropped(t *testing.T) {
	dir := t.TempDir()
	// A user file that tries to add a bypass rule plus a benign credential rule.
	yaml := `rules:
  - id: user-bypass-attempt
    category: bypass
    reason: user tries to hijack bypass
    text: 'harmless-marker-xyz'
  - id: user-credential-extra
    category: credential_access
    reason: extra secret path
    text: 'my-secret-marker-abc'
`
	if err := os.WriteFile(filepath.Join(dir, "user.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	rules, err := loadUserRulesFrom(dir)
	if err != nil {
		t.Fatalf("loadUserRulesFrom: %v", err)
	}
	for _, r := range rules {
		if r.category == CategoryBypass {
			t.Fatalf("user bypass rule was not dropped: %q", r.id)
		}
	}
	// The non-bypass user rule survives.
	found := false
	for _, r := range rules {
		if r.id == "user-credential-extra" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected non-bypass user rule to load")
	}
}

func TestEngineUserBypassNonOverridable(t *testing.T) {
	dir := t.TempDir()
	yaml := `rules:
  - id: user-bypass-attempt
    category: bypass
    text: 'harmless-marker-xyz'
`
	if err := os.WriteFile(filepath.Join(dir, "user.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngineWithUserDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	// The user's would-be bypass marker must not be denied — the rule was dropped.
	ir, _, _ := shell.Prepare("echo harmless-marker-xyz")
	res := eng.Evaluate(ir, policy.Input{Command: "echo harmless-marker-xyz"})
	for _, c := range res.Categories {
		if c == CategoryBypass {
			t.Fatalf("dropped user bypass rule still fired: %+v", res)
		}
	}
}

func TestInvalidUserYAMLSkipped(t *testing.T) {
	dir := t.TempDir()
	// One invalid file (bad regex) and one valid file: valid must still load.
	bad := `rules:
  - id: bad
    category: destructive
    argv0: '([invalid'
`
	good := `rules:
  - id: good-user
    category: credential_access
    text: 'good-marker'
`
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good.yaml"), []byte(good), 0o600); err != nil {
		t.Fatal(err)
	}

	rules, err := loadUserRulesFrom(dir)
	if err != nil {
		t.Fatalf("loadUserRulesFrom returned error, expected skip: %v", err)
	}
	found := false
	for _, r := range rules {
		if r.id == "good-user" {
			found = true
		}
		if r.id == "bad" {
			t.Fatal("invalid rule should have been skipped")
		}
	}
	if !found {
		t.Fatal("valid user rule not loaded after skipping invalid file")
	}
}

func TestMissingUserDirNotError(t *testing.T) {
	rules, err := loadUserRulesFrom(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if rules != nil {
		t.Fatalf("expected nil rules for missing dir, got %d", len(rules))
	}
}
