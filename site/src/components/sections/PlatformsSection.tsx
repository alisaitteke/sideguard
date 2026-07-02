/**
 * Supported platforms copy with badges and GitHub Releases link.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
import { Badge } from "@/components/ui/badge"

export function PlatformsSection() {
  return (
    <section id="platforms" className="py-10">
      <div className="mx-auto max-w-[52rem] px-4">
        <h2 className="mb-6 text-xl font-semibold">Supported platforms</h2>
        <p className="text-muted-foreground leading-relaxed">
          <Badge className="mr-1">macOS</Badge> and{" "}
          <Badge className="mr-1">Linux</Badge> (amd64 / arm64) are supported by
          the installer. <Badge variant="outline">Windows</Badge> is not supported
          by{" "}
          <code className="rounded bg-muted px-1 py-0.5 text-sm text-primary">
            setup.sh
          </code>{" "}
          — download the{" "}
          <code className="rounded bg-muted px-1 py-0.5 text-sm text-primary">
            .zip
          </code>{" "}
          from{" "}
          <a
            href="https://github.com/alisaitteke/sideguard/releases"
            className="text-primary hover:underline"
            rel="noopener noreferrer"
            target="_blank"
          >
            GitHub Releases
          </a>{" "}
          manually.
        </p>
      </div>
    </section>
  )
}
