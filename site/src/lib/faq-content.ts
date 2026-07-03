/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * FAQ copy shared by FaqSection and FAQPage JSON-LD for SEO / AI crawlers.
 */
export const FAQ_ITEMS = [
  {
    id: "antivirus",
    question: "Is SideGuard an antivirus or MCP malware scanner?",
    answer:
      "No. SideGuard is a local approval daemon for shell commands and MCP tools. It does not scan files, signatures, or MCP traffic with an LLM. It intercepts actions, applies your YAML policy, and waits for you to approve or deny, fail-closed by design.",
  },
  {
    id: "mcp-guard",
    question: "What is an MCP guard and how does SideGuard work?",
    answer:
      "An MCP guard intercepts Model Context Protocol tool calls before they run on your machine. SideGuard wraps MCP servers, evaluates YAML policy, and queues risky tool invocations for human approval. It is a human-in-the-loop MCP guard for Cursor and Claude Code—not a signature-based MCP antivirus.",
  },
  {
    id: "mcp-firewall",
    question: "How is SideGuard different from MCP firewalls or MCP-only scanners?",
    answer:
      "Tools like MCP-Defender proxy and scan MCP traffic. SideGuard takes a different approach: human-in-the-loop approval with YAML policy for both shell commands and MCP tool calls. It stops risky actions and asks you. It does not replace your MCP stack with a scanning proxy.",
  },
  {
    id: "vibe-coding-tools",
    question: "What vibe coding security tools should I use with Cursor or Claude Code?",
    answer:
      "For vibe coding with AI agents, pair your editor with guardrails that block shell commands and MCP tools until you approve. SideGuard is a vibe coding security tool: fail-closed hooks, YAML policy, local audit history, and MCP wrap—without cloud lock-in or passive scanning.",
  },
  {
    id: "vibe-coding-tips",
    question: "What are practical vibe coding tips for safer AI-assisted development?",
    answer:
      "Use fail-closed hooks so agents cannot run destructive commands when your guard daemon is down. Scope auto-allow rules per repo, deny reads of .ssh and .env by default, and review MCP tool calls—not just terminal commands. SideGuard encodes these vibe coding tips as YAML policy plus an approval queue.",
  },
  {
    id: "cursor-claude",
    question: "Does SideGuard work with Cursor and Claude Code?",
    answer:
      "Yes. sideguard install wires fail-closed hooks into Cursor and Claude Code, wraps MCP servers, and starts the local daemon. Use sideguard clients reload after config changes to refresh hooks without restarting your editor.",
  },
  {
    id: "approval-fatigue",
    question: "Won’t I approve everything just to move faster?",
    answer:
      "That’s approval fatigue. SideGuard addresses it with auto-allow rules for trusted commands, repo-scoped dev policies, and smart triage. Risky patterns (destructive deletes, curl|sh, secret paths) stay on deny-or-ask; routine work can flow with rules you define.",
  },
  {
    id: "fail-closed",
    question: "What does fail-closed mean for Cursor hooks?",
    answer:
      "If SideGuard’s daemon is not running, or you deny a request, the shell command or MCP tool call does not execute. Cursor’s built-in “ask” permission can still allow execution in some cases. SideGuard hooks block at the OS level until you explicitly allow.",
  },
  {
    id: "policy",
    question: "How does MCP tool approval policy work?",
    answer:
      "You write YAML rules: auto-allow, auto-deny, or ask for shell patterns and MCP tools. Policies can be scoped per repo. The daemon evaluates every intercepted action against policy before it hits your approval queue.",
  },
  {
    id: "prompt-injection",
    question: "Can prompt injection in GitHub issues affect my coding agent?",
    answer:
      "Yes. Hidden instructions in issues, README files, or comments can steer an LLM to exfiltrate secrets. SideGuard does not read issue text—it blocks the resulting shell commands and MCP tool calls until you approve, so injected instructions cannot run silently.",
  },
  {
    id: "local",
    question: "Does my code leave my machine?",
    answer:
      "No cloud required. SideGuard runs as a local daemon on loopback. History, policy, and approvals stay on your machine, built for local AI coding agent security without sending commands to a third party.",
  },
  {
    id: "install",
    question: "How do I install SideGuard?",
    answer:
      "Run curl -fsSL https://sideguard.io/setup.sh | sh on macOS or Linux, then sideguard install to register hooks and MCP wrap. One Go binary, terminal-first. See the Install section on sideguard.io for the exact command.",
  },
] as const

export type FaqItem = (typeof FAQ_ITEMS)[number]
