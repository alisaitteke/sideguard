package shell

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// maxNestDepth bounds recursive parsing of `-c`/`eval`/`env` payloads and
// command substitutions so a maliciously deep input cannot exhaust the stack.
const maxNestDepth = 5

// clusteredShortFlag matches a single-dash token of two or more letters with no
// value (e.g. "-rf"), which is canonicalized into individual "-r" "-f" flags so
// detect rules can match each flag independently.
var clusteredShortFlag = regexp.MustCompile(`^-[A-Za-z]{2,}$`)

// interpreterCommands are shells whose `-c <script>` argument is a nested
// command line that must be unwrapped and analyzed on its own.
var interpreterCommands = map[string]struct{}{
	"sh": {}, "bash": {}, "dash": {}, "zsh": {}, "ksh": {}, "ash": {},
}

// Parse normalizes the command, parses it with mvdan.cc/sh, and walks the AST
// into an IR. It never executes anything. On a parse error it returns a partial
// IR (Raw populated) plus the error so callers can still regex-match the raw
// string. Interpreter one-liners (`bash -c`, `eval`, `env … cmd`) and command
// substitutions are unwrapped into NestedCommands.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-1.0-shell.md).
func Parse(command string) (IR, error) {
	normalized := Normalize(command)
	ir := IR{Raw: normalized}
	if strings.TrimSpace(normalized) == "" {
		return ir, fmt.Errorf("shell: empty command")
	}

	file, err := parseTree(normalized)
	if err != nil {
		return ir, fmt.Errorf("shell: parse: %w", err)
	}

	buildIR(&ir, file.Stmts, 0)
	return ir, nil
}

// parseTree runs the mvdan.cc/sh parser on a source string. It is the single
// point where parsing happens, reused by Parse and by the deobfuscator to
// validate that a decoded layer still looks shell-like.
func parseTree(src string) (*syntax.File, error) {
	parser := syntax.NewParser()
	return parser.Parse(strings.NewReader(src), "")
}

// buildIR walks statements, filling stages, redirects, assignments,
// substitutions, and nested commands. depth guards recursion into nested
// interpreters and substitutions.
func buildIR(ir *IR, stmts []*syntax.Stmt, depth int) {
	for _, stmt := range stmts {
		if stmt == nil {
			continue
		}
		collectRedirects(ir, stmt.Redirs)
		collectFromCommand(ir, stmt.Cmd, depth)
	}
	if len(ir.Stages) > 0 && ir.Argv0 == "" {
		ir.Argv0 = ir.Stages[0].Argv0
		ir.Args = ir.Stages[0].Args
	}
}

// collectFromCommand dispatches on the command node kind, flattening pipelines
// and descending into subshells/blocks so every simple command becomes a Stage.
func collectFromCommand(ir *IR, cmd syntax.Command, depth int) {
	switch c := cmd.(type) {
	case nil:
		return
	case *syntax.CallExpr:
		collectCall(ir, c, depth)
	case *syntax.BinaryCmd:
		// Covers pipelines (| |&) and lists (&& ||); both flatten left→right.
		collectFromStmt(ir, c.X, depth)
		collectFromStmt(ir, c.Y, depth)
	case *syntax.Subshell:
		buildIR(ir, c.Stmts, depth)
	case *syntax.Block:
		buildIR(ir, c.Stmts, depth)
	}
}

func collectFromStmt(ir *IR, stmt *syntax.Stmt, depth int) {
	if stmt == nil {
		return
	}
	collectRedirects(ir, stmt.Redirs)
	collectFromCommand(ir, stmt.Cmd, depth)
}

// collectCall turns a simple command into a Stage, records assignments and
// substitutions, and unwraps interpreter one-liners into NestedCommands.
func collectCall(ir *IR, call *syntax.CallExpr, depth int) {
	for _, assign := range call.Assigns {
		ir.Assignments = append(ir.Assignments, assignText(assign))
	}

	// Extract any command substitutions from every argument word first, so a
	// `$(…)` inside an argument is surfaced even when the outer command is benign.
	for _, w := range call.Args {
		collectSubstitutions(ir, w, depth)
	}

	if len(call.Args) == 0 {
		return
	}

	argv := make([]string, 0, len(call.Args))
	for _, w := range call.Args {
		argv = append(argv, wordText(w))
	}

	argv0 := path.Base(argv[0])
	rest := normalizeArgs(argv[1:])
	ir.Stages = append(ir.Stages, Stage{Argv0: argv0, Args: rest})

	unwrapInterpreter(ir, argv0, argv[1:], depth)
}

// unwrapInterpreter parses the payload of `bash -c <script>`, `eval <script>`,
// and `env [VAR=v]… cmd args` into NestedCommands so hidden intent is analyzed.
func unwrapInterpreter(ir *IR, argv0 string, args []string, depth int) {
	if depth >= maxNestDepth {
		return
	}
	switch {
	case isInterpreter(argv0):
		if script, ok := dashCScript(args); ok {
			appendNested(ir, script, depth)
		}
	case argv0 == "eval":
		if len(args) > 0 {
			appendNested(ir, strings.Join(args, " "), depth)
		}
	case argv0 == "env":
		if rest := envCommand(args); rest != "" {
			appendNested(ir, rest, depth)
		}
	}
}

// appendNested parses a nested command line and appends its IR (depth-bounded).
func appendNested(ir *IR, src string, depth int) {
	nested := IR{Raw: Normalize(src)}
	if file, err := parseTree(nested.Raw); err == nil {
		buildIR(&nested, file.Stmts, depth+1)
	}
	ir.NestedCommands = append(ir.NestedCommands, nested)
}

// dashCScript returns the script argument that follows a `-c` flag, if present.
func dashCScript(args []string) (string, bool) {
	for i, a := range args {
		if a == "-c" && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// envCommand returns the command portion of `env`'s arguments, skipping leading
// flags and VAR=value assignments.
func envCommand(args []string) string {
	i := 0
	for i < len(args) {
		a := args[i]
		if strings.HasPrefix(a, "-") || strings.Contains(a, "=") {
			i++
			continue
		}
		break
	}
	if i >= len(args) {
		return ""
	}
	return strings.Join(args[i:], " ")
}

// collectSubstitutions walks a word for command substitutions ($()/backticks),
// recording each body as text and parsing it into a NestedCommand.
func collectSubstitutions(ir *IR, node syntax.Node, depth int) {
	if node == nil {
		return
	}
	syntax.Walk(node, func(n syntax.Node) bool {
		if cs, ok := n.(*syntax.CmdSubst); ok {
			body := nodesToString(cs.Stmts)
			if body != "" {
				ir.Substitutions = append(ir.Substitutions, body)
				if depth < maxNestDepth {
					appendNested(ir, body, depth)
				}
			}
		}
		return true
	})
}

func collectRedirects(ir *IR, redirs []*syntax.Redirect) {
	for _, r := range redirs {
		if r == nil {
			continue
		}
		red := Redirect{Op: r.Op.String()}
		if r.Word != nil {
			red.Target = wordText(r.Word)
		}
		ir.Redirects = append(ir.Redirects, red)
	}
}

// normalizeArgs expands clustered short flags (`-rf` → `-r`, `-f`) so detect
// rules can match individual flags. Non-clustered tokens pass through unchanged.
func normalizeArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if clusteredShortFlag.MatchString(a) {
			for _, r := range a[1:] {
				out = append(out, "-"+string(r))
			}
			continue
		}
		out = append(out, a)
	}
	return out
}

func isInterpreter(argv0 string) bool {
	_, ok := interpreterCommands[argv0]
	return ok
}

// wordText renders a word to plain text for detection: literal and quoted
// segments are unquoted; parameter expansions and other dynamic parts are
// rendered from source so their surface form is still visible.
func wordText(w *syntax.Word) string {
	if w == nil {
		return ""
	}
	var b strings.Builder
	for _, part := range w.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			b.WriteString(p.Value)
		case *syntax.SglQuoted:
			b.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, inner := range p.Parts {
				if lit, ok := inner.(*syntax.Lit); ok {
					b.WriteString(lit.Value)
				} else {
					b.WriteString(nodeToString(inner))
				}
			}
		default:
			b.WriteString(nodeToString(part))
		}
	}
	return b.String()
}

func assignText(a *syntax.Assign) string {
	if a == nil || a.Name == nil {
		return ""
	}
	value := ""
	if a.Value != nil {
		value = wordText(a.Value)
	}
	return a.Name.Value + "=" + value
}

// nodeToString prints a single AST node back to source text.
func nodeToString(node syntax.Node) string {
	var b strings.Builder
	if err := syntax.NewPrinter().Print(&b, node); err != nil {
		return ""
	}
	return strings.TrimSpace(b.String())
}

// nodesToString prints a list of statements back to a single-line source string.
func nodesToString(stmts []*syntax.Stmt) string {
	parts := make([]string, 0, len(stmts))
	for _, st := range stmts {
		if st == nil {
			continue
		}
		if s := nodeToString(st); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "; ")
}
