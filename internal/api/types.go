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
