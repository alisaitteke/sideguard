// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package detect

import (
	"strings"

	"github.com/alisaitteke/sideguard/internal/policy"
	"github.com/alisaitteke/sideguard/internal/shell"
)

// maxUnwrap bounds how many levels of interpreter_escape NestedCommands the
// engine re-evaluates, so a deeply wrapped payload cannot cause unbounded
// recursion. Two iterations cover realistic `bash -c "python -c …"` nesting.
const maxUnwrap = 2

// Result is the outcome of detect evaluation. It carries the policy Action so it
// slots into the existing evaluate pipeline (Phase 3), plus the matched rule
// ids, an advisory numeric score, and the categories that fired for audit.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-2.0-detect.md).
type Result struct {
	// Action is the decision (allow/deny/ask) per the scoring contract.
	Action policy.Action
	// Reason is a human-readable explanation for the decision.
	Reason string
	// MatchedRules lists the ids of every rule that fired (deduplicated).
	MatchedRules []string
	// Score is an advisory aggregate risk weight (higher = riskier).
	Score int
	// Categories lists the distinct categories that matched.
	Categories []Category
}

// Engine holds the compiled embedded + user rule set and evaluates shell.IR
// against it. It is safe for concurrent read use after construction; all regexes
// are compiled once by NewEngine. It never touches the network or executes
// commands — evaluation is pure pattern matching over the static IR.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-2.0-detect.md).
type Engine struct {
	rules []compiledRule
}

// NewEngine builds an Engine from the embedded rule packs plus any user rule
// packs in ~/.sideguard/rules. Embedded rules load first; user rules are merged
// after and may add allow/deny/ask signals, but user-supplied bypass rules are
// dropped at load (SideGuard self-protection is non-overridable).
func NewEngine() (*Engine, error) {
	embedded, err := loadEmbedded()
	if err != nil {
		return nil, err
	}
	user, err := loadUserRules()
	if err != nil {
		return nil, err
	}
	return &Engine{rules: append(embedded, user...)}, nil
}

// NewEngineWithUserDir is NewEngine with an explicit user rules directory,
// exposed for tests that need a controlled ~/.sideguard/rules equivalent.
func NewEngineWithUserDir(userDir string) (*Engine, error) {
	embedded, err := loadEmbedded()
	if err != nil {
		return nil, err
	}
	user, err := loadUserRulesFrom(userDir)
	if err != nil {
		return nil, err
	}
	return &Engine{rules: append(embedded, user...)}, nil
}

// Evaluate scans the IR against every rule and returns the aggregate decision.
// interpreter_escape matches cause the IR's NestedCommands to be re-evaluated
// (bounded by maxUnwrap) so hidden intent behind `bash -c`/`python -c` counts.
func (e *Engine) Evaluate(ir shell.IR, input policy.Input) Result {
	matches := e.collect(ir, input.ToolName, 0)
	matches = dedupeMatches(matches)

	action, score, reason := decide(matches, ir.Argv0)

	return Result{
		Action:       action,
		Reason:       reason,
		MatchedRules: matchIDs(matches),
		Score:        score,
		Categories:   matchCategories(matches),
	}
}

// collect gathers rule matches for one IR and recurses into NestedCommands when
// an interpreter_escape rule fired, up to maxUnwrap levels deep.
func (e *Engine) collect(ir shell.IR, toolName string, depth int) []ruleMatch {
	matches := e.matchRules(ir, toolName)

	if depth < maxUnwrap && hasCategory(matches, CategoryInterpreterEscape) {
		for _, nested := range ir.NestedCommands {
			matches = append(matches, e.collect(nested, toolName, depth+1)...)
		}
	}
	return matches
}

// matchRules evaluates every compiled rule against a single IR (no recursion).
func (e *Engine) matchRules(ir shell.IR, toolName string) []ruleMatch {
	haystack := buildHaystack(ir)
	var out []ruleMatch
	for _, r := range e.rules {
		if ruleMatches(r, ir, haystack, toolName) {
			out = append(out, ruleMatch{
				id:       r.id,
				category: r.category,
				severity: r.severity,
				reason:   r.reason,
			})
		}
	}
	return out
}

// ruleMatches reports whether every matcher set on a rule is satisfied. argv0
// and args (when both set) must match the SAME pipeline stage; text matches a
// flattened view of the whole command; mcp_tool matches the tool name.
func ruleMatches(r compiledRule, ir shell.IR, haystack, toolName string) bool {
	if r.mcpRe != nil && !r.mcpRe.MatchString(toolName) {
		return false
	}
	if r.textRe != nil && !r.textRe.MatchString(haystack) {
		return false
	}
	if r.argv0Re != nil || r.argsRe != nil {
		if !stageMatches(r, ir.Stages) {
			return false
		}
	}
	return true
}

// stageMatches reports whether any stage satisfies both the argv0 and args
// matchers that are set on the rule.
func stageMatches(r compiledRule, stages []shell.Stage) bool {
	for _, st := range stages {
		if r.argv0Re != nil && !r.argv0Re.MatchString(st.Argv0) {
			continue
		}
		if r.argsRe != nil && !r.argsRe.MatchString(strings.Join(st.Args, " ")) {
			continue
		}
		return true
	}
	return false
}

// buildHaystack flattens an IR into a single searchable string: the normalized
// raw command, every stage's argv0+args, and every command-substitution body.
// This lets text-matcher rules (e.g. path-based bypass/credential rules) fire
// regardless of which structural field holds the offending token.
func buildHaystack(ir shell.IR) string {
	var b strings.Builder
	b.WriteString(ir.Raw)
	for _, st := range ir.Stages {
		b.WriteByte(' ')
		b.WriteString(st.Argv0)
		if len(st.Args) > 0 {
			b.WriteByte(' ')
			b.WriteString(strings.Join(st.Args, " "))
		}
	}
	for _, s := range ir.Substitutions {
		b.WriteByte(' ')
		b.WriteString(s)
	}
	return b.String()
}

// hasCategory reports whether any match belongs to category c.
func hasCategory(matches []ruleMatch, c Category) bool {
	for _, m := range matches {
		if m.category == c {
			return true
		}
	}
	return false
}

// dedupeMatches removes duplicate rule ids (a rule may match both a parent IR
// and a nested one) so scoring counts each rule at most once.
func dedupeMatches(matches []ruleMatch) []ruleMatch {
	seen := make(map[string]struct{}, len(matches))
	out := make([]ruleMatch, 0, len(matches))
	for _, m := range matches {
		if _, ok := seen[m.id]; ok {
			continue
		}
		seen[m.id] = struct{}{}
		out = append(out, m)
	}
	return out
}

// matchIDs returns the ordered rule ids from a match set.
func matchIDs(matches []ruleMatch) []string {
	if len(matches) == 0 {
		return nil
	}
	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		ids = append(ids, m.id)
	}
	return ids
}

// matchCategories returns the distinct categories from a match set, in order.
func matchCategories(matches []ruleMatch) []Category {
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[Category]struct{}, len(matches))
	out := make([]Category, 0, len(matches))
	for _, m := range matches {
		if _, ok := seen[m.category]; ok {
			continue
		}
		seen[m.category] = struct{}{}
		out = append(out, m.category)
	}
	return out
}
