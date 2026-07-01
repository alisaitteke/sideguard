// Package hook implements the Cursor/Claude shell and MCP hook bridge.
// Blocking stdin/stdout normalization with daemon long-poll approval.
// See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-5.0-hook-bridge.md).
package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/llm"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

const (
	// ExitAllow is returned when the user approves the action.
	ExitAllow = 0
	// ExitDeny follows Cursor hook convention for blocked actions.
	ExitDeny = 2
)

// RunShell handles beforeShellExecution (Cursor) and PreToolUse Bash (Claude).
func RunShell(stdin io.Reader, stdout io.Writer, client DaemonClient) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		_ = cursorDeny(stdout, "failed to read hook input", "hook stdin read error")
		return ExitDeny
	}

	if shellHookPassThrough(data) {
		return respondAllow(stdout, detectShellHookFormat(data) == formatClaude)
	}

	claude := detectShellHookFormat(data) == formatClaude
	return runApproval(stdout, client, claude, func() (api.ApprovalRequest, error) {
		if claude {
			command, cwd, err := parseClaudeShell(data)
			if err != nil {
				return api.ApprovalRequest{}, err
			}
			return api.ApprovalRequest{
				Source:  "shell",
				Client:  detectClient("claude"),
				Command: command,
				CWD:     cwd,
			}, nil
		}

		command, cwd, err := parseCursorShell(data)
		if err != nil {
			return api.ApprovalRequest{}, err
		}
		return api.ApprovalRequest{
			Source:  "shell",
			Client:  detectClient("cursor"),
			Command: command,
			CWD:     cwd,
		}, nil
	})
}

// RunMCP handles beforeMCPExecution (Cursor) and PreToolUse mcp__* (Claude).
// When both hook and MCP proxy fire, each creates a separate approval (no dedupe in v1).
func RunMCP(stdin io.Reader, stdout io.Writer, client DaemonClient) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		_ = cursorDeny(stdout, "failed to read hook input", "hook stdin read error")
		return ExitDeny
	}

	claude := detectMCPHookFormat(data) == formatClaude
	return runApproval(stdout, client, claude, func() (api.ApprovalRequest, error) {
		if claude {
			toolName, toolInput, cwd, err := parseClaudeMCP(data)
			if err != nil {
				return api.ApprovalRequest{}, err
			}
			return api.ApprovalRequest{
				Source:    "mcp",
				Client:    detectClient("claude"),
				Command:   fmt.Sprintf("mcp:%s", toolName),
				CWD:       cwd,
				ToolName:  toolName,
				ToolInput: toolInput,
			}, nil
		}

		toolName, toolInput, cwd, err := parseCursorMCP(data)
		if err != nil {
			return api.ApprovalRequest{}, err
		}
		return api.ApprovalRequest{
			Source:    "mcp",
			Client:    detectClient("cursor"),
			Command:   fmt.Sprintf("mcp:%s", toolName),
			CWD:       cwd,
			ToolName:  toolName,
			ToolInput: toolInput,
		}, nil
	})
}

func runApproval(stdout io.Writer, client DaemonClient, claude bool, build func() (api.ApprovalRequest, error)) int {
	req, err := build()
	if err != nil {
		if claude {
			_ = claudeDeny(stdout, err.Error())
		} else {
			_ = cursorDeny(stdout, err.Error(), "invalid hook input")
		}
		return ExitDeny
	}

	if devBypassEnabled() {
		return respondAllow(stdout, claude)
	}
	if req.Source == "shell" && policy.IsControlPlaneCommand(req.Command) {
		return respondAllow(stdout, claude)
	}

	policyInput := policy.Input{
		Command:  req.Command,
		ToolName: req.ToolName,
		CWD:      req.CWD,
	}
	llmEnabled := llm.Enabled(req.CWD)
	clf, clfErr := llm.ClassifierFor(req.CWD)
	if clfErr != nil {
		log.Printf("vibeguard llm: classifier init failed (fail-safe ask): %v", clfErr)
	}
	policyResult := policy.EvaluateWithLLM(context.Background(), req.CWD, policyInput, clf, llmEnabled)
	switch policyResult.Action {
	case policy.ActionAllow:
		return respondAllow(stdout, claude)
	case policy.ActionDeny:
		reason := policyResult.Reason
		if reason == "" {
			reason = "blocked by policy"
		}
		if claude {
			_ = claudeDeny(stdout, reason)
		} else {
			_ = cursorDeny(stdout, reason, reason)
		}
		return ExitDeny
	}

	ctx := context.Background()
	decision, err := client.RequestAndWait(ctx, req)
	if err != nil {
		if claude {
			_ = claudeDenyUnreachable(stdout)
		} else {
			_ = cursorDenyUnreachable(stdout)
		}
		return ExitDeny
	}

	if decision.Permission == "allow" {
		return respondAllow(stdout, claude)
	}

	if claude {
		reason := decision.AgentMessage
		if reason == "" {
			reason = decision.UserMessage
		}
		_ = claudeDeny(stdout, reason)
	} else {
		_ = cursorDeny(stdout, decision.UserMessage, decision.AgentMessage)
	}
	return ExitDeny
}

func decodeToolInput(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]any{}, nil
	}

	var asObject map[string]any
	if err := json.Unmarshal(raw, &asObject); err == nil {
		return asObject, nil
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if asString == "" {
			return map[string]any{}, nil
		}
		var nested map[string]any
		if err := json.Unmarshal([]byte(asString), &nested); err == nil {
			return nested, nil
		}
		return map[string]any{"raw": asString}, nil
	}

	return nil, fmt.Errorf("unsupported tool_input format")
}

func respondAllow(stdout io.Writer, claude bool) int {
	if claude {
		_ = claudeAllow(stdout)
	} else {
		_ = cursorAllow(stdout)
	}
	return ExitAllow
}

// devBypassEnabled is true when VIBEGUARD_DEV=1 or VIBEGUARD_BYPASS=1 (local dev/testing only).
func devBypassEnabled() bool {
	for _, key := range []string{"VIBEGUARD_DEV", "VIBEGUARD_BYPASS"} {
		switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
		case "1", "true", "yes":
			return true
		}
	}
	return false
}

func detectClient(fallback string) string {
	if os.Getenv("CURSOR_TRACE_ID") != "" || os.Getenv("CURSOR_SESSION_ID") != "" {
		return "cursor"
	}
	if os.Getenv("CLAUDE_CODE") != "" || os.Getenv("CLAUDECODE") != "" {
		return "claude"
	}
	return fallback
}
