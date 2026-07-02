package policy

// DefaultTemplate is the starter policy written on install when no file exists.
// Mirrors docs/roadmap.md example policy.
const DefaultTemplate = `rules:
  - match: { command: '^sideguard (pending|approve|deny|ui|status|daemon|uninstall|doctor|policy|clients)(\s|$)' }
    action: allow
    reason: "SideGuard control-plane CLI (unblocks hook deadlock)"
  - match: { command: "^git (status|diff|log)" }
    action: allow
  - match: { command: "^(curl|wget) " }
    action: ask
  - match: { mcp_tool: ".*delete.*" }
    action: deny
    reason: "Destructive MCP tools blocked"
`
