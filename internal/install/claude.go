package install

import (
	"encoding/json"
	"fmt"
	"os"
)

type claudeHookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

type claudeHookMatcher struct {
	Matcher string              `json:"matcher"`
	Hooks   []claudeHookCommand `json:"hooks"`
}

type claudeSettingsDoc struct {
	Hooks map[string][]claudeHookMatcher `json:"hooks,omitempty"`
}

// PatchClaudeMCP rewrites mcpServers in ~/.claude.json.
func PatchClaudeMCP(path, binary string, dryRun bool) (changed int, diff string, err error) {
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

// PatchClaudeHooks merges PreToolUse hooks for Bash and MCP tools.
func PatchClaudeHooks(path, binary string, dryRun bool) (added int, diff string, err error) {
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
	doc := &claudeSettingsDoc{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, doc); err != nil {
			return 0, "", fmt.Errorf("parse claude settings.json: %w", err)
		}
	}
	if doc.Hooks == nil {
		doc.Hooks = map[string][]claudeHookMatcher{}
	}

	shellCmd := absoluteHookCommand(binary, "shell")
	mcpCmd := absoluteHookCommand(binary, "mcp")

	beforeCount := countClaudeHookCommands(doc.Hooks["PreToolUse"])
	doc.Hooks["PreToolUse"] = mergeClaudePreToolUse(doc.Hooks["PreToolUse"], shellCmd, mcpCmd)
	afterCount := countClaudeHookCommands(doc.Hooks["PreToolUse"])
	added = afterCount - beforeCount
	if added < 0 {
		added = 0
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
	return added, diffSummary(path, before, string(out)), nil
}

func mergeClaudePreToolUse(existing []claudeHookMatcher, shellCmd, mcpCmd string) []claudeHookMatcher {
	existing = mergeClaudeMatcher(existing, "Bash", shellCmd)
	existing = mergeClaudeMatcher(existing, "mcp__.*", mcpCmd)
	return existing
}

func mergeClaudeMatcher(existing []claudeHookMatcher, matcher, command string) []claudeHookMatcher {
	entry := claudeHookCommand{Type: "command", Command: command, Timeout: 600}

	for i, m := range existing {
		if m.Matcher != matcher {
			continue
		}
		for _, h := range m.Hooks {
			if hookCommandsEqual(h.Command, command) {
				return existing
			}
		}
		existing[i].Hooks = append(existing[i].Hooks, entry)
		return existing
	}

	return append(existing, claudeHookMatcher{
		Matcher: matcher,
		Hooks:   []claudeHookCommand{entry},
	})
}

func countClaudeHookCommands(matchers []claudeHookMatcher) int {
	n := 0
	for _, m := range matchers {
		n += len(m.Hooks)
	}
	return n
}

func absoluteHookCommand(binary, kind string) string {
	suffix := " hook " + kind
	if binary == "" {
		return "vibeguard" + suffix
	}
	return binary + suffix
}
