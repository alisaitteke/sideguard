package shell

import (
	"strings"
	"testing"
)

func TestNormalizeStripsZeroWidth(t *testing.T) {
	in := "c\u200burl\u200d evil.example"
	got := Normalize(in)
	if got != "curl evil.example" {
		t.Fatalf("normalize = %q, want %q", got, "curl evil.example")
	}
}

func TestNormalizeNFKC(t *testing.T) {
	// Fullwidth "ｒｍ" (U+FF52 U+FF4D) normalizes to ASCII "rm" under NFKC.
	in := "\uFF52\uFF4D -rf /tmp"
	got := Normalize(in)
	if !strings.HasPrefix(got, "rm ") {
		t.Fatalf("normalize NFKC = %q, want prefix %q", got, "rm ")
	}
}

func TestPrepareSurfacesBase64Payload(t *testing.T) {
	// Manual check from the phase spec: base64-in-substitution should surface rm.
	ir, meta, err := Prepare("echo $(echo cm0gLXJmIC8= | base64 -d)")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if meta.Depth < 1 || !containsString(meta.Layers, "base64") {
		t.Fatalf("expected base64 decode, meta=%+v", meta)
	}
	if !irContains(ir, "rm") {
		t.Fatalf("expected rm surfaced, ir=%+v", ir)
	}
}

func TestPrepareRawIsNormalizedOriginal(t *testing.T) {
	cmd := "echo $(echo cm0gLXJmIC8= | base64 -d)"
	ir, _, err := Prepare(cmd)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	// Raw must be the normalized ORIGINAL, not the deobfuscated (appended) string.
	if ir.Raw != cmd {
		t.Fatalf("Raw = %q, want %q", ir.Raw, cmd)
	}
	if strings.Contains(ir.Raw, "rm -rf") {
		t.Fatalf("Raw should not contain appended decoded payload: %q", ir.Raw)
	}
}

func TestPrepareEmptyCommand(t *testing.T) {
	_, _, err := Prepare("")
	if err == nil {
		t.Fatalf("expected error for empty command")
	}
}

func TestPrepareNoExecutionSideEffects(t *testing.T) {
	// A destructive payload must be analyzed statically, never run. We cannot
	// observe execution directly, but we assert Prepare returns structured data
	// and does not block/panic on a dangerous decoded command.
	ir, _, err := Prepare("bash -c \"$(echo cm0gLXJmIC8= | base64 -d)\"")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if len(ir.Stages) == 0 {
		t.Fatalf("expected stages, got none")
	}
}
