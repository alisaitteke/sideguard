package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// CursorPermissionResponse is stdout for Cursor beforeShellExecution / beforeMCPExecution.
type CursorPermissionResponse struct {
	Permission   string `json:"permission"`
	UserMessage  string `json:"user_message,omitempty"`
	AgentMessage string `json:"agent_message,omitempty"`
}

// cursorShellInput is stdin for Cursor beforeShellExecution and preToolUse Shell.
// Newer Cursor builds may send command inside tool_input instead of the top-level field.
type cursorShellInput struct {
	Command   string          `json:"command"`
	CWD       string          `json:"cwd"`
	Sandbox   bool            `json:"sandbox"`
	HookName  string          `json:"hook_event_name"`
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// cursorMCPInput is stdin for Cursor beforeMCPExecution.
type cursorMCPInput struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	CWD       string          `json:"cwd"`
	HookName  string          `json:"hook_event_name"`
}

func parseCursorShell(data []byte) (command, cwd string, err error) {
	var in cursorShellInput
	if err := json.Unmarshal(data, &in); err != nil {
		return "", "", fmt.Errorf("parse cursor shell hook input: %w", err)
	}

	command = strings.TrimSpace(in.Command)
	var toolInput map[string]any
	if command == "" && len(in.ToolInput) > 0 {
		toolInput, err = decodeToolInput(in.ToolInput)
		if err != nil {
			return "", "", err
		}
		command = shellCommandFromToolInput(toolInput)
	}
	if command == "" {
		return "", "", fmt.Errorf("missing command in hook input")
	}

	cwd = strings.TrimSpace(in.CWD)
	if cwd == "" && toolInput != nil {
		if wd, ok := toolInput["working_directory"].(string); ok {
			cwd = strings.TrimSpace(wd)
		}
	}
	if cwd == "" {
		cwd = "."
	}
	return command, cwd, nil
}

func shellCommandFromToolInput(toolInput map[string]any) string {
	if toolInput == nil {
		return ""
	}
	for _, key := range []string{"command", "cmd"} {
		if raw, ok := toolInput[key]; ok {
			if s, ok := raw.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func parseCursorMCP(data []byte) (toolName string, toolInput map[string]any, cwd string, err error) {
	var in cursorMCPInput
	if err := json.Unmarshal(data, &in); err != nil {
		return "", nil, "", fmt.Errorf("parse cursor mcp hook input: %w", err)
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

func writeCursorResponse(w io.Writer, permission, userMessage, agentMessage string) error {
	out := CursorPermissionResponse{
		Permission:   permission,
		UserMessage:  userMessage,
		AgentMessage: agentMessage,
	}
	enc := json.NewEncoder(w)
	return enc.Encode(out)
}

func cursorDenyUnreachable(w io.Writer) error {
	return writeCursorResponse(w, "deny",
		"SideGuard daemon is not running. Start with: sideguard daemon start",
		"Run sideguard ui to review.")
}

func cursorDeny(w io.Writer, userMessage, agentMessage string) error {
	if agentMessage == "" {
		agentMessage = "Run sideguard ui to review."
	}
	return writeCursorResponse(w, "deny", userMessage, agentMessage)
}

func cursorAllow(w io.Writer) error {
	return writeCursorResponse(w, "allow", "", "")
}
