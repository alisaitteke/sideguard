/**
 * Landing page footer: GitHub, license, and domain links.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
export function SiteFooter() {
  return (
    <footer className="mt-auto border-t border-border py-8">
      <div className="mx-auto max-w-[52rem] space-y-1 px-4 text-center text-sm text-muted-foreground">
        <p>
          <a
            href="https://github.com/alisaitteke/sideguard"
            className="text-primary hover:underline"
            rel="noopener noreferrer"
            target="_blank"
          >
            github.com/alisaitteke/sideguard
          </a>
        </p>
        <p className="text-muted-foreground/80">
          Local approval daemon · shell + MCP guardrails · MIT License
        </p>
        <p>
          <a
            href="https://sideguard.io"
            className="text-primary hover:underline"
          >
            sideguard.io
          </a>
        </p>
      </div>
    </footer>
  )
}
