// Package install discovers client configs and wires hooks/MCP wraps.
// See docs/plans/2026-07-01-0127-sideguard-foundation/ (vgf-phase-3.0-install.md).
package install

import (
	"os"
	"path/filepath"
)

// Client identifies which AI coding agent a config belongs to.
type Client string

const (
	ClientCursor Client = "cursor"
	ClientClaude Client = "claude"
)

// ConfigKind classifies a discovered config file.
type ConfigKind string

const (
	KindMCP   ConfigKind = "mcp"
	KindHooks ConfigKind = "hooks"
)

// Target is a config file SideGuard can patch during install.
type Target struct {
	Client Client
	Kind   ConfigKind
	Path   string
}

// DiscoverOptions controls which clients and scopes are scanned.
type DiscoverOptions struct {
	Cursor bool
	Claude bool
	Cwd    string // project directory; empty uses os.Getwd()
}

// Discover scans well-known Cursor and Claude Code config paths.
// User-global paths are always checked; project paths under Cwd are included when present.
func Discover(opts DiscoverOptions) ([]Target, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cwd := opts.Cwd
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	var targets []Target

	if opts.Cursor {
		targets = append(targets,
			Target{Client: ClientCursor, Kind: KindMCP, Path: filepath.Join(home, ".cursor", "mcp.json")},
			Target{Client: ClientCursor, Kind: KindHooks, Path: filepath.Join(home, ".cursor", "hooks.json")},
			Target{Client: ClientCursor, Kind: KindMCP, Path: filepath.Join(cwd, ".cursor", "mcp.json")},
			Target{Client: ClientCursor, Kind: KindHooks, Path: filepath.Join(cwd, ".cursor", "hooks.json")},
		)
	}

	if opts.Claude {
		targets = append(targets,
			Target{Client: ClientClaude, Kind: KindMCP, Path: filepath.Join(home, ".claude.json")},
			Target{Client: ClientClaude, Kind: KindHooks, Path: filepath.Join(home, ".claude", "settings.json")},
			Target{Client: ClientClaude, Kind: KindMCP, Path: filepath.Join(cwd, ".mcp.json")},
			Target{Client: ClientClaude, Kind: KindHooks, Path: filepath.Join(cwd, ".claude", "settings.json")},
		)
	}

	seen := make(map[string]struct{}, len(targets))
	var out []Target
	for _, t := range targets {
		if _, ok := seen[t.Path]; ok {
			continue
		}
		seen[t.Path] = struct{}{}
		if _, err := os.Stat(t.Path); err == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

// ListMCPServers returns human-readable MCP server names found in discover targets.
func ListMCPServers(targets []Target) ([]string, error) {
	var names []string
	seen := make(map[string]struct{})

	for _, t := range targets {
		if t.Kind != KindMCP {
			continue
		}
		servers, err := readMCPServerNames(t)
		if err != nil {
			return nil, err
		}
		for _, name := range servers {
			key := string(t.Client) + ":" + name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			names = append(names, string(t.Client)+": "+name)
		}
	}
	return names, nil
}

func readMCPServerNames(t Target) ([]string, error) {
	data, err := os.ReadFile(t.Path)
	if err != nil {
		return nil, err
	}

	switch t.Client {
	case ClientCursor:
		doc, err := parseCursorMCP(data)
		if err != nil {
			return nil, err
		}
		return mcpServerNames(doc.MCPServers), nil
	case ClientClaude:
		doc, err := parseClaudeJSON(data)
		if err != nil {
			return nil, err
		}
		return mcpServerNames(doc.MCPServers), nil
	default:
		return nil, nil
	}
}

func mcpServerNames(servers map[string]mcpServerEntry) []string {
	if len(servers) == 0 {
		return nil
	}
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	return names
}
