/**
 * FAQ section: SEO-friendly answers on positioning, Cursor/Claude support, and policy.
 */
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"

const FAQ_ITEMS = [
  {
    id: "antivirus",
    question: "Is SideGuard an antivirus or MCP malware scanner?",
    answer:
      "No. SideGuard is a local approval daemon for shell commands and MCP tools. It does not scan files, signatures, or MCP traffic with an LLM. It intercepts actions, applies your YAML policy, and waits for you to approve or deny, fail-closed by design.",
  },
  {
    id: "mcp-firewall",
    question: "How is this different from MCP firewalls or MCP-only scanners?",
    answer:
      "Tools like MCP-Defender proxy and scan MCP traffic. SideGuard takes a different approach: human-in-the-loop approval with YAML policy for both shell commands and MCP tool calls. It stops risky actions and asks you. It does not replace your MCP stack with a scanning proxy.",
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
    id: "local",
    question: "Does my code leave my machine?",
    answer:
      "No cloud required. SideGuard runs as a local daemon on loopback. History, policy, and approvals stay on your machine, built for local AI coding agent security without sending commands to a third party.",
  },
  {
    id: "install",
    question: "How do I install SideGuard?",
    answer:
      "Run curl -fsSL https://sideguard.io/setup.sh | sh on macOS or Linux, then sideguard install to register hooks and MCP wrap. One Go binary, terminal-first. See the Install section below for the exact command.",
  },
] as const

export function FaqSection() {
  return (
    <section id="faq" className="py-10">
      <div className="mx-auto max-w-[52rem] px-4">
        <h2 className="mb-2 text-xl font-semibold">Frequently asked questions</h2>
        <p className="mb-6 text-muted-foreground">
          AI agent command approval, MCP tool policy, and how SideGuard fits your
          workflow.
        </p>
        <Accordion>
          {FAQ_ITEMS.map((item) => (
            <AccordionItem key={item.id} value={item.id}>
              <AccordionTrigger>{item.question}</AccordionTrigger>
              <AccordionContent>
                <p>{item.answer}</p>
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </section>
  )
}
