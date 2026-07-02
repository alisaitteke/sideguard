/**
 * Honest positioning callout: what SideGuard is not (SEO clarity, not defensive).
 */
export function NotWhatSection() {
  return (
    <section
      id="not-antivirus"
      aria-label="What SideGuard is not"
      className="py-6"
    >
      <div className="mx-auto max-w-[52rem] px-4">
        <div className="rounded-xl border border-border bg-muted/30 px-4 py-4 sm:px-6">
          <p className="text-sm leading-relaxed text-muted-foreground">
            <span className="font-medium text-foreground">
              What SideGuard is not:
            </span>{" "}
            not an MCP antivirus, not an MCP malware scanner, and not an AI
            firewall that proxies your traffic. It does not scan tool payloads
            with signatures or LLMs. It stops risky actions and asks you to
            approve. Beyond MCP-only scanners, SideGuard guards shell commands{" "}
            <em>and</em> MCP tools with YAML policy you control.
          </p>
        </div>
      </div>
    </section>
  )
}
