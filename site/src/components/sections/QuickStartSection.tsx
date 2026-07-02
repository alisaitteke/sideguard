/**
 * Post-install quick start CLI commands.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
import { QUICK_START_COMMANDS } from "@/lib/install"

export function QuickStartSection() {
  return (
    <section id="quickstart" className="py-10">
      <div className="mx-auto max-w-[52rem] px-4">
        <h2 className="mb-6 text-xl font-semibold">Quick start</h2>
        <p className="mb-4 text-muted-foreground">
          After the binary is installed, wire SideGuard into your AI clients:
        </p>
        <pre className="overflow-x-auto rounded-lg border border-border bg-card p-4 font-mono text-sm leading-relaxed">
          <code>{QUICK_START_COMMANDS}</code>
        </pre>
      </div>
    </section>
  )
}
