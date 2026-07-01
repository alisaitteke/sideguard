package hook

import (
	"encoding/json"
	"fmt"
	"io"
)

// CursorPermissionResponse is stdout for Cursor beforeShellExecution / beforeMCPExecution.
type CursorPermissionResponse struct {
	Permission   string `json:"permission"`
	UserMessage  string `json:"user_message,omitempty"`
	AgentMessage string `json:"agent_message,omitempty"`
}

// cursorShellInput is stdin for Cursor beforeShellExecution.
type cursorShellInput struct {
	Command  string `json:"command"`
	CWD      string `json:"cwd"`
	Sandbox  bool   `json:"sandbox"`
	HookName string `json:"hook_event_name"`
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
	if in.Command == "" {
		return "", "", fmt.Errorf("missing command in hook input")
	}
	cwd = in.CWD
	if cwd == "" {
		cwd = "."
	}
	return in.Command, cwd, nil
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
		"VibeGuard daemon is not running. Start with: vibeguard daemon start",
		"Run vibeguard ui to review.")
}

func cursorDeny(w io.Writer, userMessage, agentMessage string) error {
	if agentMessage == "" {
		agentMessage = "Run vibeguard ui to review."
	}
	return writeCursorResponse(w, "deny", userMessage, agentMessage)
}

func cursorAllow(w io.Writer) error {
	return writeCursorResponse(w, "allow", "", "")
}
