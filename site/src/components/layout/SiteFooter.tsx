/**
 * Landing page footer: GitHub, license, domain, and subtle creator attribution.
 * Plan: portfolio subtle attribution (Tier 3).
 */
import {
  AUTHOR_LINKEDIN_URL,
  AUTHOR_NAME,
  AUTHOR_SITE_URL,
} from "@/lib/author"

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
          Vibe coding security tool · MCP guard · shell + MCP approval · MIT
          License
        </p>
        <p>
          <a
            href="https://sideguard.io"
            className="text-primary hover:underline"
          >
            sideguard.io
          </a>
        </p>
        <p className="pt-2 text-xs text-muted-foreground/70">
          Built by{" "}
          <a
            href={AUTHOR_SITE_URL}
            className="text-primary hover:underline"
            rel="noopener noreferrer"
            target="_blank"
          >
            {AUTHOR_NAME}
          </a>
          {" · "}
          <a
            href={AUTHOR_LINKEDIN_URL}
            className="text-primary hover:underline"
            rel="noopener noreferrer"
            target="_blank"
          >
            LinkedIn
          </a>
        </p>
      </div>
    </footer>
  )
}
