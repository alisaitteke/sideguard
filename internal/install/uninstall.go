package install

import (
	"fmt"
	"os"
	"runtime"

	"github.com/alisaitteke/sideguard/internal/daemon"
	"github.com/alisaitteke/sideguard/internal/tray"
)

// UninstallResult summarizes uninstall actions for CLI output.
// See docs/plans/2026-07-01-1418-uninstall-architecture/ (uia-phase-1.0-install-uninstall.md).
type UninstallResult struct {
	Mode          string
	FilesChanged  []string
	HooksRemoved  int
	MCPUnwrapped  int
	DaemonRemoved bool
	TrayRemoved   bool
	Warnings      []string
}

// Uninstall removes SideGuard hooks, MCP wraps, and optionally the daemon LaunchAgent.
// Default mode is surgical in-place removal; --restore-backup uses the oldest backup per file.
func Uninstall(opts Options) (*UninstallResult, error) {
	if !opts.Cursor && !opts.Claude {
		opts.Cursor = true
		opts.Claude = true
	}

	result := &UninstallResult{}
	if opts.RestoreBackup {
		result.Mode = "restore-backup"
		if err := uninstallRestoreBackup(opts, result); err != nil {
			return nil, err
		}
	} else {
		result.Mode = "surgical"
		if err := uninstallSurgical(opts, result); err != nil {
			return nil, err
		}
	}

	if !opts.KeepDaemon && !opts.DryRun {
		if err := daemon.UninstallService(); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("daemon: %v", err))
		} else {
			result.DaemonRemoved = true
		}
		if runtime.GOOS == "darwin" {
			if err := tray.UninstallService(); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("tray: %v", err))
			} else {
				result.TrayRemoved = true
			}
		}
	}

	printUninstallSummary(result, opts)
	return result, nil
}

func uninstallRestoreBackup(opts Options, result *UninstallResult) error {
	paths := uninstallPaths(opts)
	if opts.DryRun {
		for _, p := range paths {
			if _, err := os.Stat(p); err != nil {
				continue
			}
			session, _, err := findFirstBackup(p)
			if err != nil {
				return err
			}
			if session != "" {
				result.FilesChanged = append(result.FilesChanged, p)
			}
		}
		return nil
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		session, rel, err := findFirstBackup(p)
		if err != nil {
			return err
		}
		if session == "" {
			continue
		}
		if err := restoreFromSession(session, rel, p); err != nil {
			return err
		}
		result.FilesChanged = append(result.FilesChanged, p)
	}
	return nil
}

func uninstallSurgical(opts Options, result *UninstallResult) error {
	binary, err := resolveBinary()
	if err != nil {
		return err
	}

	targets, err := Discover(DiscoverOptions{
		Cursor: opts.Cursor,
		Claude: opts.Claude,
		Cwd:    opts.Cwd,
	})
	if err != nil {
		return err
	}

	seen := make(map[string]bool)
	for _, t := range targets {
		if seen[t.Path] {
			continue
		}
		seen[t.Path] = true

		changed, hooks, mcp, err := unpatchTarget(t, binary, opts.DryRun)
		if err != nil {
			return err
		}
		result.HooksRemoved += hooks
		result.MCPUnwrapped += mcp
		if changed {
			result.FilesChanged = append(result.FilesChanged, t.Path)
		}
	}
	return nil
}

func unpatchTarget(t Target, binary string, dryRun bool) (changed bool, hooksRemoved, mcpUnwrapped int, err error) {
	switch t.Kind {
	case KindMCP:
		var n int
		var diff string
		switch t.Client {
		case ClientCursor:
			n, diff, err = UnpatchCursorMCP(t.Path, binary, dryRun)
		case ClientClaude:
			n, diff, err = UnpatchClaudeMCP(t.Path, binary, dryRun)
		}
		if err != nil {
			return false, 0, 0, err
		}
		return diff != "" || n > 0, 0, n, nil
	case KindHooks:
		var n int
		var diff string
		switch t.Client {
		case ClientCursor:
			n, diff, err = UnpatchCursorHooks(t.Path, binary, dryRun)
		case ClientClaude:
			n, diff, err = UnpatchClaudeHooks(t.Path, binary, dryRun)
		}
		if err != nil {
			return false, 0, 0, err
		}
		return diff != "" || n > 0, n, 0, nil
	default:
		return false, 0, 0, nil
	}
}

func uninstallPaths(opts Options) []string {
	return appendHookPaths(nil, opts)
}

func printUninstallSummary(result *UninstallResult, opts Options) {
	prefix := "Uninstall"
	if opts.DryRun {
		prefix = "Dry-run uninstall"
	}
	fmt.Printf("\n%s summary (mode: %s)\n", prefix, result.Mode)
	fmt.Printf("  hooks removed: %d\n", result.HooksRemoved)
	fmt.Printf("  MCP servers unwrapped: %d\n", result.MCPUnwrapped)
	if result.DaemonRemoved {
		fmt.Println("  daemon LaunchAgent removed")
	}
	if result.TrayRemoved {
		fmt.Println("  tray LaunchAgent removed")
	}
	if len(result.FilesChanged) > 0 {
		fmt.Println("  files changed:")
		for _, f := range result.FilesChanged {
			fmt.Printf("    - %s\n", f)
		}
	}
	for _, w := range result.Warnings {
		fmt.Printf("  warning: %s\n", w)
	}
	if !opts.DryRun {
		PrintClientReloadHints(opts, "uninstall changes", ReloadHintsBrief)
	}
}
