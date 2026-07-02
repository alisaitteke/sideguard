package hook

import (
	"encoding/json"
	"fmt"
	"io"
)

// ClaudeHookResponse is stdout for Claude Code PreToolUse command hooks.
type ClaudeHookResponse struct {
	HookSpecificOutput ClaudeHookDecision `json:"hookSpecificOutput"`
}

// ClaudeHookDecision carries the PreToolUse permission decision.
type ClaudeHookDecision struct {
	HookEventName              string `json:"hookEventName"`
	PermissionDecision         string `json:"permissionDecision"`
	PermissionDecisionReason   string `json:"permissionDecisionReason,omitempty"`
}

// claudePreToolUseInput is stdin for Claude PreToolUse hooks (Bash and MCP).
type claudePreToolUseInput struct {
	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	CWD           string          `json:"cwd"`
}


func parseClaudeShell(data []byte) (command, cwd string, err error) {
	var in claudePreToolUseInput
	if err := json.Unmarshal(data, &in); err != nil {
		return "", "", fmt.Errorf("parse claude pretooluse input: %w", err)
	}

	toolInput, err := decodeToolInput(in.ToolInput)
	if err != nil {
		return "", "", err
	}

	command, _ = toolInput["command"].(string)
	if command == "" {
		return "", "", fmt.Errorf("missing bash command in tool_input")
	}

	cwd = in.CWD
	if cwd == "" {
		cwd = "."
	}
	return command, cwd, nil
}

func parseClaudeMCP(data []byte) (toolName string, toolInput map[string]any, cwd string, err error) {
	var in claudePreToolUseInput
	if err := json.Unmarshal(data, &in); err != nil {
		return "", nil, "", fmt.Errorf("parse claude pretooluse input: %w", err)
	}
	if in.ToolName == "" {
		return "", nil, "", fmt.Errorf("missing tool_name in hook input")
	}

	toolInput, err = decodeToolInput(in.ToolInput)
	if err != nil {
		return "", nil, "", err
	}

	cwd = in.CWD
	if cwd == "" {
		cwd = "."
	}
	return in.ToolName, toolInput, cwd, nil
}

func writeClaudeResponse(w io.Writer, decision, reason string) error {
	out := ClaudeHookResponse{
		HookSpecificOutput: ClaudeHookDecision{
			HookEventName:            "PreToolUse",
			PermissionDecision:       decision,
			PermissionDecisionReason: reason,
		},
	}
	enc := json.NewEncoder(w)
	return enc.Encode(out)
}

func claudeDenyUnreachable(w io.Writer) error {
	return writeClaudeResponse(w, "deny",
		"SideGuard daemon is not running. Start with: sideguard daemon start")
}

func claudeDeny(w io.Writer, reason string) error {
	if reason == "" {
		reason = "Run sideguard ui to review."
	}
	return writeClaudeResponse(w, "deny", reason)
}

func claudeAllow(w io.Writer) error {
	return writeClaudeResponse(w, "allow", "")
}
