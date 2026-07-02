/**
 * Problem → solution feature cards grounded in vibe-coding security pain points.
 */
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

const FEATURES = [
  {
    problem: "Dangerous shell commands run without asking",
    solution:
      "Fail-closed Cursor hooks block rm -rf, curl|sh, and force push before they execute. Review and approve in the terminal UI or macOS tray.",
  },
  {
    problem: "MCP tool poisoning in hidden instructions",
    solution:
      "Every MCP tool call goes through SideGuard wrap and an approval queue; no server is trusted blindly.",
  },
  {
    problem: "Credential and secret exfiltration",
    solution:
      "YAML policy auto-denies reads of .ssh, .env, and other sensitive paths. You define what never leaves your machine.",
  },
  {
    problem: "Approval fatigue slows you down",
    solution:
      "Auto-allow rules, smart triage, and repo-scoped dev policies cut noise while keeping fail-closed defaults for risky actions.",
  },
  {
    problem: "No visibility into what agents ran",
    solution: (
      <>
        <code className="rounded bg-muted px-1 py-0.5 text-sm text-primary">
          sideguard history
        </code>{" "}
        gives you a local audit trail of shell commands and MCP tool calls:
        approved, denied, and pending.
      </>
    ),
  },
  {
    problem: "Cursor “ask” permission doesn’t actually block",
    solution:
      "SideGuard is a real fail-closed daemon on loopback. If the daemon is down or you deny, the command does not run.",
  },
  {
    problem: "MCP servers trusted blindly",
    solution:
      "MCP wrap routes every tool invocation through policy and your approval queue, not straight to the host.",
  },
  {
    problem: "Security tooling that’s hard to install",
    solution: (
      <>
        One{" "}
        <code className="rounded bg-muted px-1 py-0.5 text-sm text-primary">
          curl setup.sh
        </code>
        , a single Go binary, terminal-first workflow, wired into Cursor and
        Claude Code with{" "}
        <code className="rounded bg-muted px-1 py-0.5 text-sm text-primary">
          sideguard install
        </code>
        .
      </>
    ),
  },
] as const

export function FeaturesSection() {
  return (
    <section id="features" className="py-10">
      <div className="mx-auto max-w-[52rem] px-4">
        <h2 className="mb-2 text-xl font-semibold">
          Vibe coding security guardrails that actually block
        </h2>
        <p className="mb-6 text-muted-foreground">
          Local AI coding agent security for Cursor and Claude Code: policy-driven
          command approval, not passive scanning.
        </p>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {FEATURES.map((feature) => (
            <Card key={feature.problem}>
              <CardHeader>
                <CardDescription className="text-xs uppercase tracking-wide">
                  Problem
                </CardDescription>
                <CardTitle className="text-base">{feature.problem}</CardTitle>
              </CardHeader>
              <CardContent className="text-muted-foreground">
                <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-foreground/70">
                  SideGuard
                </span>
                {feature.solution}
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </section>
  )
}
