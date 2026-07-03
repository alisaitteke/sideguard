// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// mcpServerEntry is the shared STDIO MCP server shape across clients.
type mcpServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

type cursorMCPDoc struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

type claudeJSONDoc struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

func parseCursorMCP(data []byte) (*cursorMCPDoc, error) {
	var doc cursorMCPDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse cursor mcp.json: %w", err)
	}
	if doc.MCPServers == nil {
		doc.MCPServers = map[string]mcpServerEntry{}
	}
	return &doc, nil
}

func parseClaudeJSON(data []byte) (*claudeJSONDoc, error) {
	var doc claudeJSONDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse claude.json: %w", err)
	}
	if doc.MCPServers == nil {
		doc.MCPServers = map[string]mcpServerEntry{}
	}
	return &doc, nil
}

func unwrapMCPServers(servers map[string]mcpServerEntry, binary string) int {
	if len(servers) == 0 {
		return 0
	}

	unwrapped := 0
	for name, entry := range servers {
		for isAlreadyWrapped(entry, binary) {
			upstreamCmd := entry.Args[2]
			upstreamArgs := append([]string{}, entry.Args[3:]...)
			entry.Command = upstreamCmd
			entry.Args = upstreamArgs
			unwrapped++
		}
		servers[name] = entry
	}
	return unwrapped
}

func unpatchMCPServersJSON(data []byte, binary string) ([]byte, int, error) {
	var unwrapped int
	out, err := patchJSONObject(data, func(doc map[string]json.RawMessage) error {
		servers, err := rawMCPServers(doc)
		if err != nil {
			return err
		}
		unwrapped = unwrapMCPServers(servers, binary)
		return setRawMCPServers(doc, servers)
	})
	return out, unwrapped, err
}

// UnpatchCursorMCP removes sideguard wrap -- from STDIO MCP servers in-place.
func UnpatchCursorMCP(path, binary string, dryRun bool) (unwrapped int, diff string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, "", nil
		}
		return 0, "", err
	}

	out, unwrapped, err := unpatchMCPServersJSON(data, binary)
	if err != nil {
		return 0, "", err
	}
	if string(data) == string(out) {
		return 0, "", nil
	}

	if err := writeFileAtomic(path, out, dryRun); err != nil {
		return 0, "", err
	}
	return unwrapped, diffSummary(path, string(data), string(out)), nil
}

// UnpatchClaudeMCP removes sideguard wrap -- from mcpServers in ~/.claude.json.
func UnpatchClaudeMCP(path, binary string, dryRun bool) (unwrapped int, diff string, err error) {
	return UnpatchCursorMCP(path, binary, dryRun)
}

func wrapMCPServers(servers map[string]mcpServerEntry, binary string) (int, error) {
	if len(servers) == 0 {
		return 0, nil
	}

	wrapped := 0
	for name, entry := range servers {
		if entry.URL != "" {
			continue
		}
		if isAlreadyWrapped(entry, binary) {
			continue
		}
		if entry.Command == "" {
			continue
		}

		upstreamArgs := append([]string{}, entry.Args...)
		entry.Args = append([]string{"wrap", "--", entry.Command}, upstreamArgs...)
		entry.Command = binary
		servers[name] = entry
		wrapped++
	}
	return wrapped, nil
}

func isAlreadyWrapped(entry mcpServerEntry, binary string) bool {
	if !commandIsSideguard(entry.Command, binary) {
		return false
	}
	return len(entry.Args) >= 2 && entry.Args[0] == "wrap" && entry.Args[1] == "--"
}

func commandIsSideguard(command, binary string) bool {
	if isGuardBinaryName(command) {
		return true
	}
	if binary != "" && command == binary {
		return true
	}
	return isGuardBinaryName(filepath.Base(command))
}

func isGuardBinaryName(name string) bool {
	switch name {
	case "sideguard", "vibeguard":
		return true
	default:
		return false
	}
}

func marshalJSONPretty(v any) ([]byte, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func patchMCPServersJSON(data []byte, binary string) ([]byte, int, error) {
	var wrapped int
	out, err := patchJSONObject(data, func(doc map[string]json.RawMessage) error {
		servers, err := rawMCPServers(doc)
		if err != nil {
			return err
		}
		wrapped, err = wrapMCPServers(servers, binary)
		if err != nil {
			return err
		}
		return setRawMCPServers(doc, servers)
	})
	return out, wrapped, err
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func writeFileAtomic(path string, data []byte, dryRun bool) error {
	if dryRun {
		return nil
	}
	if err := ensureParentDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func diffSummary(label, before, after string) string {
	if before == after {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s (before)\n", label)
	fmt.Fprintf(&b, "+++ %s (after)\n", label)
	fmt.Fprintf(&b, "%s\n", after)
	return b.String()
}
