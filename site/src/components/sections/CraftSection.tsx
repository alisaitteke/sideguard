/**
 * Technical craft section: architecture signals for curious visitors and recruiters.
 * Plan: portfolio subtle attribution (Tier 3).
 */
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { ARCHITECTURE_DOC_URL } from "@/lib/author"

const CRAFT_ITEMS = [
  {
    headline: "One Go binary",
    body: "Daemon on loopback HTTP, Cursor/Claude hook bridge, MCP STDIO proxy, bubbletea terminal UI, and an optional CGO macOS menu-bar tray — all in a single shipped binary.",
  },
  {
    headline: "Hybrid interception",
    body: "Shell commands via editor hooks; MCP tool calls via server wrap. Fail-closed when the daemon is unreachable — risky actions never reach the OS silently.",
  },
  {
    headline: "Policy before LLM",
    body: "YAML auto-allow and auto-deny rules plus an obfuscation-aware detect engine. LLM analyse is on-demand and provider-driven — not a black-box auto-allow.",
  },
  {
    headline: "Shipped end-to-end",
    body: "GoReleaser multi-arch releases, curl installer with checksum verification, and this Vite + React landing on GitHub Pages — product, infra, and docs in one repo.",
  },
] as const

export function CraftSection() {
  return (
    <section id="craft" className="py-10">
      <div className="mx-auto max-w-[52rem] px-4">
        <h2 className="mb-2 text-xl font-semibold">How it&apos;s built</h2>
        <p className="mb-6 text-muted-foreground">
          SideGuard is a local security layer for AI coding agents — designed
          for production use, not a demo script.
        </p>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {CRAFT_ITEMS.map((item) => (
            <Card key={item.headline}>
              <CardHeader>
                <CardDescription className="text-xs uppercase tracking-wide">
                  Architecture
                </CardDescription>
                <CardTitle className="text-base">{item.headline}</CardTitle>
              </CardHeader>
              <CardContent className="text-muted-foreground">
                {item.body}
              </CardContent>
            </Card>
          ))}
        </div>
        <p className="mt-6 text-sm text-muted-foreground">
          Full system view:{" "}
          <a
            href={ARCHITECTURE_DOC_URL}
            className="text-primary hover:underline"
            rel="noopener noreferrer"
            target="_blank"
          >
            Architecture on GitHub
          </a>
        </p>
      </div>
    </section>
  )
}
