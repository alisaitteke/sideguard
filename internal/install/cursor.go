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

// PatchCursorMCP rewrites STDIO MCP servers to vibeguard wrap -- <upstream>.
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

// PatchCursorHooks merges VibeGuard hook entries without removing user hooks.
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

func trimHookCommand(cmd string) string {
	const suffix = " hook shell"
	if len(cmd) > len(suffix) && cmd[len(cmd)-len(suffix):] == suffix {
		return "vibeguard hook shell"
	}
	const mcpSuffix = " hook mcp"
	if len(cmd) > len(mcpSuffix) && cmd[len(cmd)-len(mcpSuffix):] == mcpSuffix {
		return "vibeguard hook mcp"
	}
	return cmd
}
