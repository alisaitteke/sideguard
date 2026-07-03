// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package policy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/alisaitteke/sideguard/internal/paths"
)

// Policy is a compiled, ready-to-evaluate policy document.
type Policy struct {
	rules []compiledRule
}

type compiledRule struct {
	action    Action
	reason    string
	commandRe *regexp.Regexp
	mcpToolRe *regexp.Regexp
	pathRe    *regexp.Regexp
}

// Load reads the global policy (~/.sideguard/policy.yaml) and optional workspace
// override (.sideguard/policy.yaml under cwd). Workspace rules are appended after
// global rules; deny beats ask beats allow across all matches.
func Load(cwd string) (*Policy, error) {
	var merged File

	home, err := paths.Home()
	if err != nil {
		return nil, err
	}

	globalPath := filepath.Join(home, paths.PolicyFile)
	if data, err := os.ReadFile(globalPath); err == nil {
		var doc File
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse global policy %s: %w", globalPath, err)
		}
		merged.Rules = append(merged.Rules, doc.Rules...)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read global policy %s: %w", globalPath, err)
	}

	if cwd != "" {
		workspacePath := filepath.Join(cwd, paths.DirName, paths.PolicyFile)
		if data, err := os.ReadFile(workspacePath); err == nil {
			var doc File
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return nil, fmt.Errorf("parse workspace policy %s: %w", workspacePath, err)
			}
			merged.Rules = append(merged.Rules, doc.Rules...)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read workspace policy %s: %w", workspacePath, err)
		}
	}

	return compile(&merged)
}

// LoadFile loads and compiles a single policy file (used by validate).
func LoadFile(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc File
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse policy %s: %w", path, err)
	}
	return compile(&doc)
}

// FromRules builds a policy from in-memory rules (tests).
func FromRules(rules []Rule) (*Policy, error) {
	return compile(&File{Rules: rules})
}

func compile(doc *File) (*Policy, error) {
	p := &Policy{}
	for i, rule := range doc.Rules {
		cr, err := compileRule(rule, i)
		if err != nil {
			return nil, err
		}
		if cr != nil {
			p.rules = append(p.rules, *cr)
		}
	}
	return p, nil
}

func compileRule(rule Rule, index int) (*compiledRule, error) {
	if rule.Action != ActionAllow && rule.Action != ActionDeny && rule.Action != ActionAsk {
		return nil, fmt.Errorf("rule %d: invalid action %q", index, rule.Action)
	}

	hasMatcher := rule.Match.Command != "" || rule.Match.MCPTool != "" || rule.Match.Path != ""
	if !hasMatcher {
		return nil, fmt.Errorf("rule %d: at least one matcher required", index)
	}

	cr := &compiledRule{action: rule.Action, reason: rule.Reason}

	if rule.Match.Command != "" {
		re, err := regexp.Compile(rule.Match.Command)
		if err != nil {
			return nil, fmt.Errorf("rule %d: invalid command regex: %w", index, err)
		}
		cr.commandRe = re
	}
	if rule.Match.MCPTool != "" {
		re, err := regexp.Compile(rule.Match.MCPTool)
		if err != nil {
			return nil, fmt.Errorf("rule %d: invalid mcp_tool regex: %w", index, err)
		}
		cr.mcpToolRe = re
	}
	if rule.Match.Path != "" {
		re, err := regexp.Compile(rule.Match.Path)
		if err != nil {
			return nil, fmt.Errorf("rule %d: invalid path regex: %w", index, err)
		}
		cr.pathRe = re
	}

	return cr, nil
}

// GlobalPath returns ~/.sideguard/policy.yaml.
func GlobalPath() (string, error) {
	home, err := paths.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, paths.PolicyFile), nil
}

// EnsureDefault writes the default policy template when the global file is missing.
func EnsureDefault() (string, error) {
	path, err := GlobalPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(DefaultTemplate), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// AppendAllowRule adds an allow rule to the global policy file.
func AppendAllowRule(match Match, reason string) error {
	path, err := GlobalPath()
	if err != nil {
		return err
	}

	var doc File
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse policy: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	doc.Rules = append(doc.Rules, Rule{
		Match:  match,
		Action: ActionAllow,
		Reason: reason,
	})

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}
