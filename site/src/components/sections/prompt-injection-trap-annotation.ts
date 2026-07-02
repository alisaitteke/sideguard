import { annotate } from "rough-notation"

type RoughAnnotation = ReturnType<typeof annotate>

/** Beat after scroll lands, before Rough Notation draws. */
export const POST_SCROLL_WAIT_MS = 1_500

/** Beat after annotation, before zoom. */
export const POST_ANNOTATION_WAIT_MS = 1_500

/** Matches Rough Notation “Annotation styling” demo — hand-drawn box around the trap line. */
export const TRAP_ANNOTATION_MS = 1_000

const TRAP_ANNOTATION_CONFIG = {
  type: "box" as const,
  color: "#f85149",
  strokeWidth: 2,
  animationDuration: TRAP_ANNOTATION_MS,
  padding: [4, 6, 4, 6] as [number, number, number, number],
  multiline: true,
}

export function createTrapLineAnnotation(
  element: HTMLElement,
  animate = true
): RoughAnnotation {
  return annotate(element, {
    ...TRAP_ANNOTATION_CONFIG,
    animate,
  })
}

/**
 * Keeps Rough Notation visibility and SVG anchors in sync with the camera timeline.
 * Call `relayout()` after every metrics recompute (resize, font load).
 */
export class TrapAnnotationController {
  private annotation: RoughAnnotation | null = null

  bind(target: HTMLElement, animate = true) {
    this.dispose()
    this.annotation = createTrapLineAnnotation(target, animate)
  }

  sync(timeMs: number, showAfterMs: number) {
    if (!this.annotation) {
      return
    }

    const shouldShow = timeMs >= showAfterMs

    if (shouldShow) {
      if (!this.annotation.isShowing()) {
        this.annotation.show()
      }
    } else if (this.annotation.isShowing()) {
      this.annotation.hide()
    }
  }

  /** Redraw annotation boxes at the current DOM positions (no draw animation). */
  relayout() {
    if (!this.annotation?.isShowing()) {
      return
    }

    this.annotation.show()
  }

  dispose() {
    this.annotation?.remove()
    this.annotation = null
  }
}

/** @deprecated Use TrapAnnotationController — kept for reduced-motion static path. */
export function startTrapAnnotationSync(
  animation: Animation,
  _durationMs: number,
  showAfterMs: number,
  target: HTMLElement,
  animate = true
): () => void {
  const controller = new TrapAnnotationController()
  controller.bind(target, animate)

  let rafId = 0

  const tick = () => {
    if (animation.playState === "finished") {
      return
    }

    const time =
      typeof animation.currentTime === "number" ? animation.currentTime : 0
    controller.sync(time, showAfterMs)
    rafId = requestAnimationFrame(tick)
  }

  rafId = requestAnimationFrame(tick)

  return () => {
    cancelAnimationFrame(rafId)
    controller.dispose()
  }
}
