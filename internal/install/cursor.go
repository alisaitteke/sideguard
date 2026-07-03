// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package install

import (
	"encoding/json"
	"fmt"
	"os"
)

type cursorHookEntry struct {
	Command    string `json:"command"`
	Timeout    int    `json:"timeout,omitempty"`
	FailClosed bool   `json:"failClosed,omitempty"`
}

type cursorHooksDoc struct {
	Version int                        `json:"version,omitempty"`
	Hooks   map[string][]cursorHookEntry `json:"hooks,omitempty"`
	// Legacy flat layout (some installs omit the nested "hooks" key).
	BeforeShellExecution []cursorHookEntry `json:"beforeShellExecution,omitempty"`
	BeforeMCPExecution   []cursorHookEntry `json:"beforeMCPExecution,omitempty"`
}

// PatchCursorMCP rewrites STDIO MCP servers to sideguard wrap -- <upstream>.
func PatchCursorMCP(path, binary string, dryRun bool) (changed int, diff string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, "", err
	}

	out, wrapped, err := patchMCPServersJSON(data, binary)
	if err != nil {
		return 0, "", err
	}

	if err := writeFileAtomic(path, out, dryRun); err != nil {
		return 0, "", err
	}

	return wrapped, diffSummary(path, string(data), string(out)), nil
}

// PatchCursorHooks merges SideGuard hook entries without removing user hooks.
func PatchCursorHooks(path, binary string, dryRun bool) (added int, diff string, err error) {
	var data []byte
	if _, statErr := os.Stat(path); statErr == nil {
		data, err = os.ReadFile(path)
		if err != nil {
			return 0, "", err
		}
	} else if !os.IsNotExist(statErr) {
		return 0, "", statErr
	}

	before := string(data)
	doc, nested, err := parseCursorHooks(data)
	if err != nil {
		return 0, "", err
	}

	shellEntry := cursorHookEntry{
		Command:    absoluteHookCommand(binary, "shell"),
		Timeout:    600,
		FailClosed: true,
	}
	mcpEntry := cursorHookEntry{
		Command:    absoluteHookCommand(binary, "mcp"),
		Timeout:    600,
		FailClosed: true,
	}

	if nested {
		if doc.Hooks == nil {
			doc.Hooks = map[string][]cursorHookEntry{}
		}
		beforeCount := len(doc.Hooks["beforeShellExecution"]) + len(doc.Hooks["beforeMCPExecution"])
		doc.Hooks["beforeShellExecution"] = mergeCursorHookEntries(doc.Hooks["beforeShellExecution"], shellEntry)
		doc.Hooks["beforeMCPExecution"] = mergeCursorHookEntries(doc.Hooks["beforeMCPExecution"], mcpEntry)
		afterCount := len(doc.Hooks["beforeShellExecution"]) + len(doc.Hooks["beforeMCPExecution"])
		added = afterCount - beforeCount
		if doc.Version == 0 {
			doc.Version = 1
		}
	} else {
		beforeCount := len(doc.BeforeShellExecution) + len(doc.BeforeMCPExecution)
		doc.BeforeShellExecution = mergeCursorHookEntries(doc.BeforeShellExecution, shellEntry)
		doc.BeforeMCPExecution = mergeCursorHookEntries(doc.BeforeMCPExecution, mcpEntry)
		afterCount := len(doc.BeforeShellExecution) + len(doc.BeforeMCPExecution)
		added = afterCount - beforeCount
	}

	out, err := marshalJSONPretty(doc)
	if err != nil {
		return 0, "", err
	}

	if err := writeFileAtomic(path, out, dryRun); err != nil {
		return 0, "", err
	}

	if before == string(out) {
		return 0, "", nil
	}
	if added < 0 {
		added = 0
	}
	return added, diffSummary(path, before, string(out)), nil
}

func parseCursorHooks(data []byte) (*cursorHooksDoc, bool, error) {
	doc := &cursorHooksDoc{}
	if len(data) == 0 {
		return doc, true, nil
	}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, false, fmt.Errorf("parse cursor hooks.json: %w", err)
	}
	nested := len(doc.Hooks) > 0 || doc.Version > 0
	if !nested && (len(doc.BeforeShellExecution) > 0 || len(doc.BeforeMCPExecution) > 0) {
		return doc, false, nil
	}
	return doc, true, nil
}

func mergeCursorHookEntries(existing []cursorHookEntry, entry cursorHookEntry) []cursorHookEntry {
	for _, e := range existing {
		if hookCommandsEqual(e.Command, entry.Command) {
			return existing
		}
	}
	return append(existing, entry)
}

func hookCommandsEqual(a, b string) bool {
	if a == b {
		return true
	}
	return trimHookCommand(a) == trimHookCommand(b)
}

// IsSideguardHookCommand reports whether a hook command invokes sideguard shell or MCP hooks.
func IsSideguardHookCommand(cmd string) bool {
	t := trimHookCommand(cmd)
	return t == "sideguard hook shell" || t == "sideguard hook mcp"
}

func filterCursorHookEntries(entries []cursorHookEntry) ([]cursorHookEntry, int) {
	removed := 0
	var kept []cursorHookEntry
	for _, e := range entries {
		if IsSideguardHookCommand(e.Command) {
			removed++
			continue
		}
		kept = append(kept, e)
	}
	return kept, removed
}

// UnpatchCursorHooks removes SideGuard hook entries without touching user hooks.
func UnpatchCursorHooks(path, binary string, dryRun bool) (removed int, diff string, err error) {
	_ = binary
	var data []byte
	if _, statErr := os.Stat(path); statErr != nil {
		if os.IsNotExist(statErr) {
			return 0, "", nil
		}
		return 0, "", statErr
	}
	data, err = os.ReadFile(path)
	if err != nil {
		return 0, "", err
	}

	before := string(data)
	doc, nested, err := parseCursorHooks(data)
	if err != nil {
		return 0, "", err
	}

	if nested {
		if doc.Hooks == nil {
			return 0, "", nil
		}
		for key, entries := range doc.Hooks {
			filtered, n := filterCursorHookEntries(entries)
			removed += n
			if len(filtered) == 0 {
				delete(doc.Hooks, key)
			} else {
				doc.Hooks[key] = filtered
			}
		}
	} else {
		doc.BeforeShellExecution, removed = filterCursorHookEntries(doc.BeforeShellExecution)
		var mcpRemoved int
		doc.BeforeMCPExecution, mcpRemoved = filterCursorHookEntries(doc.BeforeMCPExecution)
		removed += mcpRemoved
	}

	out, err := marshalJSONPretty(doc)
	if err != nil {
		return 0, "", err
	}
	if before == string(out) {
		return 0, "", nil
	}

	if err := writeFileAtomic(path, out, dryRun); err != nil {
		return 0, "", err
	}
	return removed, diffSummary(path, before, string(out)), nil
}

func trimHookCommand(cmd string) string {
	const suffix = " hook shell"
	if len(cmd) > len(suffix) && cmd[len(cmd)-len(suffix):] == suffix {
		return "sideguard hook shell"
	}
	const mcpSuffix = " hook mcp"
	if len(cmd) > len(mcpSuffix) && cmd[len(cmd)-len(mcpSuffix):] == mcpSuffix {
		return "sideguard hook mcp"
	}
	return cmd
}
