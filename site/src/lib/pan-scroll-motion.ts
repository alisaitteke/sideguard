/**
 * Shared easing for pan-to-line camera moves and matching in-page scroll.
 * cubic-bezier(0.45, 0.05, 0.25, 1) — gentle ease-in at start, soft land at end.
 */
export const PAN_SCROLL_EASING = "cubic-bezier(0.45, 0.05, 0.25, 1)"

/** Camera pan segment duration (ms); page scroll uses the same value. */
export const PAN_TO_LINE_MS = 650

function createCubicBezier(x1: number, y1: number, x2: number, y2: number) {
  return (t: number): number => {
    if (t <= 0) {
      return 0
    }
    if (t >= 1) {
      return 1
    }

    let x = t

    for (let i = 0; i < 8; i++) {
      const cx =
        3 * x1 * (1 - x) * (1 - x) * x +
        3 * x2 * (1 - x) * x * x +
        x * x * x
      const dx = cx - t

      if (Math.abs(dx) < 1e-5) {
        break
      }

      const dcx =
        3 * x1 * (1 - x) * (1 - 2 * x) +
        3 * x2 * (2 * x * (1 - x) - x * x) +
        3 * x * x

      x -= dx / dcx
    }

    return (
      3 * y1 * (1 - x) * (1 - x) * x +
      3 * y2 * (1 - x) * x * x +
      x * x * x
    )
  }
}

const panScrollEase = createCubicBezier(0.45, 0.05, 0.25, 1)

/**
 * Scrolls the window to `element` using the same easing curve as the issue pan.
 */
export function smoothScrollToElement(
  element: HTMLElement,
  durationMs = PAN_TO_LINE_MS
) {
  const prefersReducedMotion = window.matchMedia(
    "(prefers-reduced-motion: reduce)"
  ).matches

  if (prefersReducedMotion) {
    element.scrollIntoView()
    return
  }

  const targetY = element.getBoundingClientRect().top + window.scrollY
  const startY = window.scrollY
  const distance = targetY - startY

  if (Math.abs(distance) < 1) {
    return
  }

  const start = performance.now()

  function step(now: number) {
    const elapsed = now - start
    const t = Math.min(elapsed / durationMs, 1)
    const eased = panScrollEase(t)

    window.scrollTo(0, startY + distance * eased)

    if (t < 1) {
      requestAnimationFrame(step)
    }
  }

  requestAnimationFrame(step)
}
