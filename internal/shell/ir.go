// Package shell is SideGuard's static shell understanding layer. It normalizes,
// deobfuscates, and parses shell command strings into a structured IR that the
// detect engine consumes. It performs pure string/AST analysis and NEVER
// executes, evals, or spawns any command.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-1.0-shell.md).
package shell

// IR is the structured, execution-free representation of a shell command.
// Every field is derived from static parsing of the (normalized, deobfuscated)
// input; nothing here is ever run. The detect engine (Phase 2) scans these
// fields instead of the raw string so obfuscated intent is surfaced.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-1.0-shell.md).
type IR struct {
	// Raw is the original command after NFKC normalization + zero-width strip.
	Raw string
	// Argv0 is the basename-normalized first token of the first stage.
	Argv0 string
	// Args are the first stage's arguments (flags/operands), post flag-normalize.
	Args []string
	// Stages are the pipeline stages, left-to-right, across all statements.
	Stages []Stage
	// Substitutions holds extracted command-substitution bodies ($(…)/`…`),
	// captured as source text only — never executed.
	Substitutions []string
	// NestedCommands holds parsed IRs unwrapped from `bash -c`/`sh -c`/`eval`/
	// `env … cmd` and from command substitutions.
	NestedCommands []IR
	// Redirects lists redirection operators and their path targets.
	Redirects []Redirect
	// Assignments lists `VAR=val` prefixes and standalone assignments.
	Assignments []string
}

// Stage is a single command within a pipeline (or a standalone statement).
type Stage struct {
	// Argv0 is the basename-normalized command name (e.g. /bin/rm → rm).
	Argv0 string
	// Args are the arguments after argv0, with clustered short flags expanded.
	Args []string
}

// Redirect is a single I/O redirection: the operator plus its target word.
// Only the textual target is captured; no file is opened.
type Redirect struct {
	// Op is the redirection operator token (e.g. ">", ">>", "<", "2>&1").
	Op string
	// Target is the redirection target as source text (usually a path).
	Target string
}

// DeobfuscateMeta records what the layered static decoder did to an input.
type DeobfuscateMeta struct {
	// Depth is the number of decode rounds applied (0 if nothing decoded).
	Depth int
	// Layers names the decoders that fired, in order (e.g. "base64", "hex").
	Layers []string
}
