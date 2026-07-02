/**
 * Feature cards grid — four SideGuard capabilities.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

const FEATURES = [
  {
    title: "Shell + MCP intercept",
    description:
      "Intercepts shell/terminal commands and MCP tool calls, holding them in an approval queue until you allow or deny.",
  },
  {
    title: "YAML policy + terminal UI",
    description: (
      <>
        Auto-allow or auto-deny via YAML rules. Review pending requests with{" "}
        <code className="rounded bg-muted px-1 py-0.5 text-sm text-primary">
          sideguard ui
        </code>{" "}
        in the terminal.
      </>
    ),
  },
  {
    title: "macOS menu-bar tray",
    description:
      "Experimental menu-bar icon for quick Allow/Deny without leaving your editor. Polls the local daemon on loopback.",
  },
  {
    title: "GitHub Releases self-update",
    description: (
      <>
        Stay current with background tray checks or run{" "}
        <code className="rounded bg-muted px-1 py-0.5 text-sm text-primary">
          sideguard update
        </code>{" "}
        from the CLI.
      </>
    ),
  },
] as const

export function FeaturesSection() {
  return (
    <section id="features" className="py-10">
      <div className="mx-auto max-w-[52rem] px-4">
        <h2 className="mb-6 text-xl font-semibold">What SideGuard does</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {FEATURES.map((feature) => (
            <Card key={feature.title}>
              <CardHeader>
                <CardTitle>{feature.title}</CardTitle>
              </CardHeader>
              <CardContent className="text-muted-foreground">
                {feature.description}
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </section>
  )
}
