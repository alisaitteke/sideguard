// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package detect

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/alisaitteke/sideguard/internal/detect/rules"
	"github.com/alisaitteke/sideguard/internal/paths"
)

// ruleFile is the on-disk YAML shape for a detect rule pack (embedded or user).
type ruleFile struct {
	Rules []ruleSpec `yaml:"rules"`
}

// ruleSpec is a single declarative detect rule. At least one matcher
// (argv0/args/text/mcp_tool) must be set; all set matchers must hold (AND).
// argv0 and args, when both set, must match the SAME pipeline stage.
type ruleSpec struct {
	ID       string   `yaml:"id"`
	Category Category `yaml:"category"`
	Severity Severity `yaml:"severity,omitempty"`
	Reason   string   `yaml:"reason,omitempty"`
	Argv0    string   `yaml:"argv0,omitempty"`
	Args     string   `yaml:"args,omitempty"`
	Text     string   `yaml:"text,omitempty"`
	MCPTool  string   `yaml:"mcp_tool,omitempty"`
}

// compiledRule is a ruleSpec with its regexes compiled once at load time.
type compiledRule struct {
	id       string
	category Category
	severity Severity
	reason   string
	argv0Re  *regexp.Regexp
	argsRe   *regexp.Regexp
	textRe   *regexp.Regexp
	mcpRe    *regexp.Regexp
}

// compileRule validates a ruleSpec and compiles its matchers. It returns
// (nil, nil) when the rule must be silently dropped (a user-supplied bypass
// rule, which is non-overridable). It returns an error on an invalid category,
// missing matcher, or an uncompilable regex.
func compileRule(spec ruleSpec, source string, allowBypass bool) (*compiledRule, error) {
	if !knownCategory(spec.Category) {
		return nil, fmt.Errorf("rule %q: unknown category %q", spec.ID, spec.Category)
	}
	// Bypass is SideGuard self-protection: only embedded packs may define it.
	if spec.Category == CategoryBypass && !allowBypass {
		log.Printf("sideguard detect: dropping non-overridable bypass rule %q from %s", spec.ID, source)
		return nil, nil
	}

	if spec.Argv0 == "" && spec.Args == "" && spec.Text == "" && spec.MCPTool == "" {
		return nil, fmt.Errorf("rule %q: at least one matcher (argv0/args/text/mcp_tool) required", spec.ID)
	}

	sev := spec.Severity
	if sev == "" {
		sev = defaultSeverity(spec.Category)
	}

	cr := &compiledRule{
		id:       spec.ID,
		category: spec.Category,
		severity: sev,
		reason:   spec.Reason,
	}

	var err error
	if spec.Argv0 != "" {
		if cr.argv0Re, err = regexp.Compile(spec.Argv0); err != nil {
			return nil, fmt.Errorf("rule %q: invalid argv0 regex: %w", spec.ID, err)
		}
	}
	if spec.Args != "" {
		if cr.argsRe, err = regexp.Compile(spec.Args); err != nil {
			return nil, fmt.Errorf("rule %q: invalid args regex: %w", spec.ID, err)
		}
	}
	if spec.Text != "" {
		if cr.textRe, err = regexp.Compile(spec.Text); err != nil {
			return nil, fmt.Errorf("rule %q: invalid text regex: %w", spec.ID, err)
		}
	}
	if spec.MCPTool != "" {
		if cr.mcpRe, err = regexp.Compile(spec.MCPTool); err != nil {
			return nil, fmt.Errorf("rule %q: invalid mcp_tool regex: %w", spec.ID, err)
		}
	}

	return cr, nil
}

// compileFile compiles all rules in one YAML document. allowBypass gates whether
// bypass-category rules are honored (true only for embedded packs).
func compileFile(data []byte, source string, allowBypass bool) ([]compiledRule, error) {
	var doc ruleFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", source, err)
	}
	out := make([]compiledRule, 0, len(doc.Rules))
	for _, spec := range doc.Rules {
		cr, err := compileRule(spec, source, allowBypass)
		if err != nil {
			return nil, err
		}
		if cr != nil {
			out = append(out, *cr)
		}
	}
	return out, nil
}

// loadEmbedded compiles the embedded rule packs. Bypass rules are honored here
// (embedded is the only trusted source for SideGuard self-protection). An
// embedded pack that fails to compile is a build-time defect and returns an error.
func loadEmbedded() ([]compiledRule, error) {
	entries, err := fs.ReadDir(rules.FS, ".")
	if err != nil {
		return nil, err
	}
	var all []compiledRule
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := rules.FS.ReadFile(e.Name())
		if err != nil {
			return nil, err
		}
		compiled, err := compileFile(data, "embedded/"+e.Name(), true)
		if err != nil {
			return nil, fmt.Errorf("embedded rule pack %s: %w", e.Name(), err)
		}
		all = append(all, compiled...)
	}
	return all, nil
}

// loadUserRules compiles user rule packs from ~/.sideguard/rules/*.yaml. It is
// best-effort: a missing directory is not an error, and a file that fails to
// parse/compile is logged and skipped so the remaining embedded rules still
// apply. User-supplied bypass rules are dropped (non-overridable).
func loadUserRules() ([]compiledRule, error) {
	dir, err := paths.RulesDir()
	if err != nil {
		return nil, err
	}
	return loadUserRulesFrom(dir)
}

// loadUserRulesFrom is loadUserRules with an explicit directory (used by tests).
func loadUserRulesFrom(dir string) ([]compiledRule, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var all []compiledRule
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("sideguard detect: skip user rule file %s: %v", path, err)
			continue
		}
		compiled, err := compileFile(data, path, false)
		if err != nil {
			log.Printf("sideguard detect: skip invalid user rule file %s: %v", path, err)
			continue
		}
		all = append(all, compiled...)
	}
	return all, nil
}
