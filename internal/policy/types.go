// Package policy implements deterministic YAML allow/deny/ask rules for shell
// commands and MCP tool names. See docs/plans/2026-07-01-0127-vibeguard-foundation/
// (vgf-phase-7.0-policy-engine.md).
package policy

// Action is the policy decision for an intercepted command or MCP tool call.
type Action string

const (
	// ActionAllow permits the operation without daemon approval.
	ActionAllow Action = "allow"
	// ActionDeny blocks the operation without notification.
	ActionDeny Action = "deny"
	// ActionAsk queues the operation for user approval.
	ActionAsk Action = "ask"
)

// File is the on-disk YAML policy document shape.
type File struct {
	Rules []Rule        `yaml:"rules"`
	LLM   *LLMFileBlock `yaml:"llm,omitempty"`
}

// LLMFileBlock is an optional per-workspace LLM override in policy.yaml.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-1.0-contracts.md).
type LLMFileBlock struct {
	Enabled *bool `yaml:"enabled,omitempty"`
}

// Rule is a single policy rule with matchers and an action.
type Rule struct {
	Match  Match  `yaml:"match"`
	Action Action `yaml:"action"`
	Reason string `yaml:"reason,omitempty"`
}

// Match holds optional regex matchers; all specified fields must match (AND).
type Match struct {
	Command string `yaml:"command,omitempty"`
	MCPTool string `yaml:"mcp_tool,omitempty"`
	Path    string `yaml:"path,omitempty"`
}

// Input is the runtime context passed to the match engine.
type Input struct {
	Command  string
	ToolName string
	CWD      string
}

// Result is the outcome of policy evaluation.
type Result struct {
	Action    Action
	Reason    string
	LoadError error
}
