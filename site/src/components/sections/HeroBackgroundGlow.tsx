/**
 * Soft top-center light beam for the landing hero (`hero-glow-*` in hero-glow.css).
 * Adapted from seatmap/web-admin-ui AppShellBackgroundGlow — scoped to the hero
 * section instead of the full viewport shell.
 */
export function HeroBackgroundGlow() {
  return (
    <div
      aria-hidden
      className="pointer-events-none absolute inset-x-0 top-[-200px] z-0 flex justify-center"
    >
      <div className="hero-glow-layer flex flex-col items-center opacity-90">
        <div className="hero-glow-core h-[420px] w-[560px] max-w-full rounded-full" />
        <div className="hero-glow-accent -mt-28 h-[280px] w-[380px] max-w-full rounded-full" />
      </div>
    </div>
  )
}
