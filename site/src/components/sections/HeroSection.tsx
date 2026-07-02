/**
 * Landing hero — logo, headline, and subhead.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
export function HeroSection() {
  return (
    <header className="py-12 text-center sm:py-16">
      <div className="mx-auto max-w-[52rem] px-4">
        <div className="mb-6 flex justify-center">
          <img
            src="/assets/logo.svg"
            alt="SideGuard"
            width={64}
            height={64}
            className="brightness-0 invert"
          />
        </div>
        <h1 className="text-2xl font-semibold leading-snug tracking-tight sm:text-3xl">
          Approve before your AI agent runs shell commands and MCP tools
        </h1>
        <p className="mt-3 text-muted-foreground">
          Cursor + Claude Code · local daemon · YAML policy
        </p>
      </div>
    </header>
  )
}
