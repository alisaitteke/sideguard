/**
 * macOS menu-bar tray: pending approvals and command history screenshots.
 */
const TRAY_SHOTS = [
  {
    src: "/assets/tray-pending-approval.png",
    alt: "SideGuard macOS tray popover showing a pending shell command with Run, Decline, and Analyse actions",
    caption: "Pending approval — review the command, tap Analyse for an LLM summary, then Run or Decline.",
  },
  {
    src: "/assets/tray-history.png",
    alt: "SideGuard macOS tray popover showing recent intercepted command history and daemon status",
    caption: "History — local audit of intercepted commands with daemon health and pending count.",
  },
] as const

export function TraySection() {
  return (
    <section
      id="tray"
      aria-labelledby="tray-heading"
      className="border-b border-border bg-background py-16 sm:py-20"
    >
      <div className="mx-auto max-w-[52rem] px-4">
        <div className="mx-auto mb-10 max-w-[38rem] text-center">
          <p className="mb-3 text-sm font-medium tracking-wide text-primary uppercase">
            macOS menu bar
          </p>
          <h2
            id="tray-heading"
            className="text-2xl font-semibold tracking-tight sm:text-3xl"
          >
            Approve from the tray — not just the terminal
          </h2>
          <p className="mt-4 text-sm leading-relaxed text-muted-foreground sm:text-base">
            Click the menu-bar shield to open a popover below the icon. Pending
            commands sit at the top with flat <strong className="font-medium text-foreground">Run</strong>{" "}
            and <strong className="font-medium text-foreground">Decline</strong>{" "}
            buttons; resolved history scrolls underneath. Approval mode, LLM
            settings, and updates live in the header menu. The tray polls the
            local daemon on loopback — nothing leaves your machine.
          </p>
        </div>

        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 sm:gap-8">
          {TRAY_SHOTS.map((shot) => (
            <figure key={shot.src} className="flex flex-col items-center">
              <div className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
                <img
                  src={shot.src}
                  alt={shot.alt}
                  width={920}
                  height={850}
                  loading="lazy"
                  className="block h-auto w-full"
                />
              </div>
              <figcaption className="mt-3 max-w-[18rem] text-center text-xs leading-relaxed text-muted-foreground sm:max-w-none sm:text-sm">
                {shot.caption}
              </figcaption>
            </figure>
          ))}
        </div>

        <p className="mx-auto mt-8 max-w-[38rem] text-center text-xs text-muted-foreground sm:text-sm">
          Experimental on macOS (CGO). Linux and Windows use a classic systray
          menu with the same pending/history layout. Terminal{" "}
          <code className="rounded bg-muted px-1 py-0.5 text-primary">
            sideguard ui
          </code>{" "}
          remains the full keyboard-driven workflow.
        </p>
      </div>
    </section>
  )
}
