// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package hook

import "encoding/json"

const (
	formatCursor = "cursor"
	formatClaude = "claude"
)

type hookProbe struct {
	HookEventName string `json:"hook_event_name"`
	Command       string `json:"command"`
	ToolName      string `json:"tool_name"`
}

func probeHookInput(data []byte) hookProbe {
	var p hookProbe
	_ = json.Unmarshal(data, &p)
	return p
}

// detectShellHookFormat chooses stdin parsing and stdout response shape for hook shell.
// Cursor beforeShellExecution includes hook_event_name but uses top-level command, not Claude tool_input.
func detectShellHookFormat(data []byte) string {
	p := probeHookInput(data)

	if p.Command != "" || p.HookEventName == "beforeShellExecution" {
		return formatCursor
	}

	if p.HookEventName == "PreToolUse" || p.ToolName == "Bash" {
		return formatClaude
	}

	// Cursor preToolUse shell tools (if wired to hook shell by mistake).
	if p.ToolName == "Shell" || p.ToolName == "run_terminal_cmd" {
		return formatCursor
	}

	return formatCursor
}

// shellHookPassThrough is true for non-shell tools that should not be gated by hook shell.
func shellHookPassThrough(data []byte) bool {
	p := probeHookInput(data)

	if p.HookEventName == "PreToolUse" && p.ToolName != "" && p.ToolName != "Bash" {
		return true
	}

	if p.HookEventName == "preToolUse" && p.ToolName != "" &&
		p.ToolName != "Shell" && p.ToolName != "run_terminal_cmd" && p.ToolName != "Bash" {
		return true
	}

	return false
}

// detectMCPHookFormat chooses stdin parsing and stdout response shape for hook mcp.
func detectMCPHookFormat(data []byte) string {
	p := probeHookInput(data)

	if p.HookEventName == "beforeMCPExecution" {
		return formatCursor
	}

	if p.HookEventName == "PreToolUse" {
		return formatClaude
	}

	// Legacy Cursor MCP payloads omit hook_event_name.
	if p.ToolName != "" {
		return formatCursor
	}

	return formatCursor
}
