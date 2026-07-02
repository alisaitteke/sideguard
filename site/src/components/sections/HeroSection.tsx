/**
 * Landing hero: value prop, tagline, and positioning for vibe coders.
 */
import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"

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
        <div className="mb-4 flex flex-wrap justify-center gap-2">
          <Badge variant="secondary">Local approval daemon</Badge>
          <Badge variant="outline">YAML policy</Badge>
          <Badge variant="outline">Fail-closed hooks</Badge>
        </div>
        <h1 className="text-2xl font-semibold leading-snug tracking-tight sm:text-3xl">
          Approve before your AI agent runs shell commands and MCP tools
        </h1>
        <p className="mx-auto mt-4 max-w-[40rem] text-base leading-relaxed text-muted-foreground sm:text-lg">
          SideGuard is human-in-the-loop security for vibe coding: a local daemon
          that intercepts risky shell commands and MCP tool calls, holds them in an
          approval queue, and blocks until you allow or deny. Policy you control,
          not an antivirus or traffic scanner.
        </p>
        <p className="mt-3 text-sm text-muted-foreground">
          Cursor shell command approval · MCP tool approval policy · Claude Code
          hook security
        </p>
        <div className="mt-6 flex flex-wrap justify-center gap-3">
          <a href="#install" className={cn(buttonVariants())}>
            Install in one command
          </a>
          <a
            href="#faq"
            className={cn(buttonVariants({ variant: "outline" }))}
          >
            Read the FAQ
          </a>
        </div>
      </div>
    </header>
  )
}
