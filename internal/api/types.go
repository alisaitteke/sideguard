// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package api

// ApprovalRequest is the payload for POST /v1/approval/request.
type ApprovalRequest struct {
	Source    string         `json:"source"`
	Client    string         `json:"client"`
	Command   string         `json:"command"`
	CWD       string         `json:"cwd"`
	ToolName  string         `json:"tool_name,omitempty"`
	ToolInput map[string]any `json:"tool_input,omitempty"`
}

// ApprovalRequestResponse is returned when a new approval is queued.
type ApprovalRequestResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// ApprovalDecision is the payload for POST /v1/approval/{id}/decide.
type ApprovalDecision struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

// ApprovalDecisionResponse is returned to hooks and MCP proxy after a decision.
type ApprovalDecisionResponse struct {
	Permission   string `json:"permission"`
	UserMessage  string `json:"user_message"`
	AgentMessage string `json:"agent_message"`
}

// PendingApproval is a summary row for GET /v1/approval/pending.
type PendingApproval struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	Client     string `json:"client"`
	Command    string `json:"command"`
	CWD        string `json:"cwd"`
	ToolName   string `json:"tool_name,omitempty"`
	CreatedAt  string `json:"created_at"`
	AgeSeconds int64  `json:"age_seconds"`
}

// HealthResponse is returned by GET /v1/health.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ApprovalModeResponse is returned by GET /v1/approval/mode.
type ApprovalModeResponse struct {
	Mode string `json:"mode"`
}

// SetApprovalModeRequest is the payload for PUT /v1/approval/mode.
type SetApprovalModeRequest struct {
	Mode string `json:"mode"`
}

// CommandEvent is the payload for POST /v1/events and rows from GET /v1/events.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-4.0-history-store.md).
type CommandEvent struct {
	ID              string   `json:"id,omitempty"`
	CreatedAt       string   `json:"created_at,omitempty"`
	Source          string   `json:"source"`
	Client          string   `json:"client"`
	CWD             string   `json:"cwd"`
	CommandRedacted string   `json:"command_redacted"`
	CommandNorm     string   `json:"command_norm"`
	ToolName        string   `json:"tool_name,omitempty"`
	YamlAction      string   `json:"yaml_action,omitempty"`
	DetectAction    string   `json:"detect_action,omitempty"`
	DetectRules     []string `json:"detect_rules,omitempty"`
	DetectScore     int      `json:"detect_score,omitempty"`
	FinalAction     string   `json:"final_action"`
	DecisionBy      string   `json:"decision_by"`
	Reason          string   `json:"reason,omitempty"`
	ApprovalID      string   `json:"approval_id,omitempty"`
	LatencyMS       int64    `json:"latency_ms,omitempty"`
}

// AnalyzeRequest is the payload for POST /v1/analyze.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-3.0-api.md).
type AnalyzeRequest struct {
	Command  string `json:"command,omitempty"`
	ToolName string `json:"tool_name,omitempty"`
	CWD      string `json:"cwd,omitempty"`
	EventID  string `json:"event_id,omitempty"`
}

// AnalyzeResponse is returned by POST /v1/analyze.
type AnalyzeResponse struct {
	Verdict      string   `json:"verdict"`
	Summary      string   `json:"summary"`
	Explanation  string   `json:"explanation"`
	Provider     string   `json:"provider"`
	DetectAction string   `json:"detect_action,omitempty"`
	DetectRules  []string `json:"detect_rules,omitempty"`
}

// EventQueryParams are GET /v1/events query parameters.
type EventQueryParams struct {
	Since  string
	Before string // RFC3339 exclusive upper bound for keyset pagination
	Denied bool
	CWD    string
	Limit  int
	Search string
}
