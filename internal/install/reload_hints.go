package install

import "fmt"

// ReloadHintsVerbosity controls how much detail PrintClientReloadHints emits.
type ReloadHintsVerbosity int

const (
	ReloadHintsBrief ReloadHintsVerbosity = iota
	ReloadHintsFull
)

// PrintClientReloadHints prints honest per-client steps to pick up hook and MCP config
// changes after install or uninstall. There is no vibeguard command that forces a reload.
func PrintClientReloadHints(opts Options, action string, verbosity ReloadHintsVerbosity) {
	cursor, claude := clientFlags(opts)
	if !cursor && !claude {
		cursor, claude = true, true
	}

	fmt.Printf("\nApply %s in your AI clients (no vibeguard reload command exists):\n", action)

	if cursor {
		printCursorReloadHints(verbosity)
	}
	if claude {
		printClaudeReloadHints(verbosity)
	}

	if verbosity == ReloadHintsBrief {
		fmt.Println("\nDetails: `vibeguard clients reload`")
	}
}

func clientFlags(opts Options) (cursor, claude bool) {
	return opts.Cursor, opts.Claude
}

func printCursorReloadHints(verbosity ReloadHintsVerbosity) {
	fmt.Println("\nCursor:")
	if verbosity == ReloadHintsFull {
		fmt.Println("  hooks (~/.cursor/hooks.json): Cursor watches hooks.json and reloads on save.")
		fmt.Println("  MCP (~/.cursor/mcp.json): wrapped servers may need the same refresh.")
	}
	fmt.Println("  Usually: save hooks.json — Cursor reloads hooks automatically.")
	fmt.Println("  If hooks or MCP wraps do not apply: Cmd+Shift+P → Developer: Reload Window")
	fmt.Println("  (reloads the window without quitting the app; lighter than a full restart).")
	if verbosity == ReloadHintsFull {
		fmt.Println("  If still broken: fully quit and reopen Cursor.")
	}
}

func printClaudeReloadHints(verbosity ReloadHintsVerbosity) {
	fmt.Println("\nClaude Code:")
	if verbosity == ReloadHintsFull {
		fmt.Println("  hooks (~/.claude/settings.json): a file watcher normally reloads hook config.")
		fmt.Println("  MCP (~/.claude.json): wrapped servers may need a session restart.")
	}
	fmt.Println("  Usually: hook changes in settings.json apply within a few seconds.")
	fmt.Println("  If hooks or MCP wraps do not apply: /exit and start a new session.")
	fmt.Println("  /hooks lists configured hooks but does not reload configuration.")
	if verbosity == ReloadHintsFull {
		fmt.Println("  There is no documented claude CLI reload command for hook settings.")
	}
}
