package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/bootstrap"
	"github.com/alisaitteke/vibeguard/internal/daemon"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

// Options controls the install wizard.
type Options struct {
	Cursor      bool
	Claude      bool
	DryRun      bool
	Discover    bool
	SkipDaemon  bool
	Cwd         string
}

// Result summarizes install actions for CLI output.
type Result struct {
	BackupDir      string
	FilesChanged   []string
	MCPServersWrap int
	HooksAdded     int
	Warnings       []string
	Diffs          []string
}

// Run executes discover → backup → MCP wrap → hook merge → daemon service install.
func Run(opts Options) (*Result, error) {
	if !opts.Cursor && !opts.Claude {
		opts.Cursor = true
		opts.Claude = true
	}

	binary, err := resolveBinary()
	if err != nil {
		return nil, err
	}

	targets, err := Discover(DiscoverOptions{
		Cursor: opts.Cursor,
		Claude: opts.Claude,
		Cwd:    opts.Cwd,
	})
	if err != nil {
		return nil, err
	}

	result := &Result{}

	if opts.Discover {
		names, err := ListMCPServers(targets)
		if err != nil {
			return nil, err
		}
		if len(names) == 0 {
			fmt.Println("No MCP servers discovered.")
		} else {
			fmt.Println("Discovered MCP servers:")
			for _, name := range names {
				fmt.Printf("  - %s\n", name)
			}
		}
		for _, t := range targets {
			fmt.Printf("  config: %s (%s %s)\n", t.Path, t.Client, t.Kind)
		}
		if opts.DryRun {
			return result, nil
		}
	}

	pathsToBackup := collectPaths(targets, opts)
	if len(pathsToBackup) == 0 {
		result.Warnings = append(result.Warnings, "no existing client configs found; will create hook files only")
	}

	// Always include hook paths we may create even when missing pre-install.
	pathsToBackup = appendHookPaths(pathsToBackup, opts)

	if !opts.DryRun && len(pathsToBackup) > 0 {
		session, err := CreateBackup(pathsToBackup)
		if err != nil {
			return nil, err
		}
		result.BackupDir = session.Dir
	}

	if err := patchTargets(targets, binary, opts.DryRun, result); err != nil {
		return nil, err
	}

	if err := ensureHookTargets(opts, binary, opts.DryRun, result); err != nil {
		return nil, err
	}

	if !opts.DryRun {
		if _, err := policy.EnsureDefault(); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("default policy: %v", err))
		}
		if err := bootstrap.EnsureDefaults(); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("default config: %v", err))
		}
		if !opts.SkipDaemon {
			if err := daemon.InstallService(); err != nil {
				return nil, fmt.Errorf("install daemon service: %w", err)
			}
		}
	}

	printSummary(result, opts.DryRun)
	return result, nil
}

// Uninstall restores backed-up configs for all known client paths.
func Uninstall(opts Options) error {
	if !opts.Cursor && !opts.Claude {
		opts.Cursor = true
		opts.Claude = true
	}

	paths := appendHookPaths(nil, opts)
	if err := RestoreLatest(paths); err != nil {
		return err
	}
	fmt.Println("Restored client configs from latest backups.")
	return nil
}

func resolveBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	return filepath.EvalSymlinks(exe)
}

func collectPaths(targets []Target, opts Options) []string {
	var paths []string
	for _, t := range targets {
		paths = append(paths, t.Path)
	}
	return uniqueStrings(paths)
}

func appendHookPaths(paths []string, opts Options) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return paths
	}
	cwd := opts.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	if opts.Cursor {
		paths = append(paths,
			filepath.Join(home, ".cursor", "hooks.json"),
			filepath.Join(cwd, ".cursor", "hooks.json"),
			filepath.Join(home, ".cursor", "mcp.json"),
			filepath.Join(cwd, ".cursor", "mcp.json"),
		)
	}
	if opts.Claude {
		paths = append(paths,
			filepath.Join(home, ".claude", "settings.json"),
			filepath.Join(cwd, ".claude", "settings.json"),
			filepath.Join(home, ".claude.json"),
			filepath.Join(cwd, ".mcp.json"),
		)
	}
	return uniqueStrings(paths)
}

func patchTargets(targets []Target, binary string, dryRun bool, result *Result) error {
	patchedMCP := make(map[string]bool)
	patchedHooks := make(map[string]bool)

	for _, t := range targets {
		switch {
		case t.Kind == KindMCP && !patchedMCP[t.Path]:
			changed, diff, err := patchMCP(t, binary, dryRun)
			if err != nil {
				return err
			}
			patchedMCP[t.Path] = true
			if changed > 0 || diff != "" {
				result.MCPServersWrap += changed
				recordChange(result, t.Path, diff)
			}
		case t.Kind == KindHooks && !patchedHooks[t.Path]:
			added, diff, err := patchHooks(t, binary, dryRun)
			if err != nil {
				return err
			}
			patchedHooks[t.Path] = true
			if added > 0 || diff != "" {
				result.HooksAdded += added
				recordChange(result, t.Path, diff)
			}
		}
	}
	return nil
}

func patchMCP(t Target, binary string, dryRun bool) (int, string, error) {
	switch t.Client {
	case ClientCursor:
		return PatchCursorMCP(t.Path, binary, dryRun)
	case ClientClaude:
		return PatchClaudeMCP(t.Path, binary, dryRun)
	default:
		return 0, "", nil
	}
}

func patchHooks(t Target, binary string, dryRun bool) (int, string, error) {
	switch t.Client {
	case ClientCursor:
		return PatchCursorHooks(t.Path, binary, dryRun)
	case ClientClaude:
		return PatchClaudeHooks(t.Path, binary, dryRun)
	default:
		return 0, "", nil
	}
}

func ensureHookTargets(opts Options, binary string, dryRun bool, result *Result) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	var hookTargets []Target
	if opts.Cursor {
		hookTargets = append(hookTargets,
			Target{Client: ClientCursor, Kind: KindHooks, Path: filepath.Join(home, ".cursor", "hooks.json")},
		)
	}
	if opts.Claude {
		hookTargets = append(hookTargets,
			Target{Client: ClientClaude, Kind: KindHooks, Path: filepath.Join(home, ".claude", "settings.json")},
		)
	}

	seen := make(map[string]bool, len(result.FilesChanged))
	for _, p := range result.FilesChanged {
		seen[p] = true
	}

	for _, t := range hookTargets {
		if seen[t.Path] {
			continue
		}
		added, diff, err := patchHooks(t, binary, dryRun)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("patch %s: %v", t.Path, err))
			continue
		}
		if added > 0 || diff != "" {
			result.HooksAdded += added
			recordChange(result, t.Path, diff)
		}
	}
	return nil
}

func recordChange(result *Result, path, diff string) {
	result.FilesChanged = append(result.FilesChanged, path)
	if diff != "" {
		result.Diffs = append(result.Diffs, diff)
	}
}

func printSummary(result *Result, dryRun bool) {
	prefix := "Install"
	if dryRun {
		prefix = "Dry-run"
	}
	fmt.Printf("\n%s summary\n", prefix)
	if result.BackupDir != "" {
		fmt.Printf("  backup: %s\n", result.BackupDir)
	}
	fmt.Printf("  MCP servers wrapped: %d\n", result.MCPServersWrap)
	fmt.Printf("  hook entries added: %d\n", result.HooksAdded)
	if len(result.FilesChanged) > 0 {
		fmt.Println("  files changed:")
		for _, f := range result.FilesChanged {
			fmt.Printf("    - %s\n", f)
		}
	}
	for _, w := range result.Warnings {
		fmt.Printf("  warning: %s\n", w)
	}
	if dryRun && len(result.Diffs) > 0 {
		fmt.Println("\nPlanned changes:")
		fmt.Println(strings.Join(result.Diffs, "\n"))
	}
	if !dryRun {
		fmt.Println("\nImportant: Cursor/Claude shell hooks now block agent commands until you approve them.")
		fmt.Println("  1. Run `vibeguard ui` for interactive approvals (or `vibeguard pending` + approve/deny for scripting).")
		fmt.Println("  2. Or open Terminal.app (outside Cursor) for approval if the agent cannot run the CLI.")
		fmt.Println("  3. Local dev/testing only: export VIBEGUARD_DEV=1 to bypass the hook queue.")
		fmt.Println("\nNext: `vibeguard status` and restart Cursor/Claude so hooks take effect.")
	}
}
