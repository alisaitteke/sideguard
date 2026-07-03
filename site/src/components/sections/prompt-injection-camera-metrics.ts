import { PAN_SCROLL_EASING } from "@/lib/pan-scroll-motion"
import {
  POST_ANNOTATION_WAIT_MS,
  POST_SCROLL_WAIT_MS,
  TRAP_ANNOTATION_MS,
} from "@/components/sections/prompt-injection-trap-annotation"

/** Fixed issue page width (matches `--gh-issue-width` in prompt-injection-scene.css). */
export const ISSUE_PAGE_WIDTH_PX = 896

/** Horizontal padding for zoom/scan framing on the fixed-width issue canvas. */
export const ISSUE_PAD_X = 40

const SCAN_PX_PER_SEC = 115
const OVERVIEW_MS = 2_000
/** Y-pan to trap line — slightly faster than shared page-scroll pan (650ms). */
const PAN_MS = 450
const HOLD_MS =
  POST_SCROLL_WAIT_MS + TRAP_ANNOTATION_MS + POST_ANNOTATION_WAIT_MS
const POST_ZOOM_WAIT_MS = 1_500
const ZOOM_MS = 1_000
const TAIL_MS = 1_200
const MIN_SCAN_MS = 1_600

/** Trap line geometry in camera-local space (camera transform must be identity). */
export type TrapLayout = {
  cmdLeft: number
  cmdRight: number
  cmdCenterY: number
  cmdHeight: number
  focusCenterX: number
  focusCenterY: number
  viewportWidth: number
  viewportHeight: number
  padX: number
}

export type CameraMetrics = {
  durationMs: number
  panY: number
  zoomX: number
  zoomY: number
  scanEndX: number
  zoomScale: number
  panEnd: number
  holdEnd: number
  zoomEnd: number
  postZoomEnd: number
  scanEnd: number
  overviewEnd: number
  annotationShowAfterMs: number
}

export function getAnnotationShowAfterMs(): number {
  return OVERVIEW_MS + PAN_MS + POST_SCROLL_WAIT_MS
}

/** Extra freeze on the last frame so GIF loops do not snap back instantly. */
const PAN_GIF_HOLD_MS = 1_400

/** Vertical pan + trap highlight — short clip for GIF export (see render-prompt-injection-video.mjs). */
export function getPanGifSegmentMs(): {
  startMs: number
  endMs: number
  holdMs: number
} {
  const startMs = OVERVIEW_MS - 500
  const endMs =
    OVERVIEW_MS + PAN_MS + POST_SCROLL_WAIT_MS + TRAP_ANNOTATION_MS + 400
  return { startMs, endMs, holdMs: PAN_GIF_HOLD_MS }
}

/**
 * Clears active transforms so layout is measured in camera-local coordinates.
 * Call before every metrics pass (initial layout, resize, font load).
 */
export function neutralizeCameraForMeasure(camera: HTMLElement): void {
  camera.getAnimations().forEach((animation) => {
    animation.cancel()
  })
  camera.style.removeProperty("transform")
  void camera.offsetHeight
}

function readTrapLayoutAtRest(
  viewport: HTMLElement,
  camera: HTMLElement,
  command: HTMLElement
): TrapLayout | null {
  const viewportRect = viewport.getBoundingClientRect()
  const cameraRect = camera.getBoundingClientRect()
  const lineRects = [...command.getClientRects()].filter((rect) => rect.width > 0)

  if (
    viewportRect.width === 0 ||
    viewportRect.height === 0 ||
    lineRects.length === 0
  ) {
    return null
  }

  let cmdLeft = Number.POSITIVE_INFINITY
  let cmdRight = Number.NEGATIVE_INFINITY
  let cmdTop = Number.POSITIVE_INFINITY
  let cmdBottom = Number.NEGATIVE_INFINITY

  for (const rect of lineRects) {
    cmdLeft = Math.min(cmdLeft, rect.left - cameraRect.left)
    cmdRight = Math.max(cmdRight, rect.right - cameraRect.left)
    cmdTop = Math.min(cmdTop, rect.top - cameraRect.top)
    cmdBottom = Math.max(cmdBottom, rect.bottom - cameraRect.top)
  }

  return {
    cmdLeft,
    cmdRight,
    cmdCenterY: (cmdTop + cmdBottom) / 2,
    cmdHeight: cmdBottom - cmdTop,
    focusCenterX:
      viewportRect.width / 2 - (cameraRect.left - viewportRect.left),
    focusCenterY:
      viewportRect.height / 2 - (cameraRect.top - viewportRect.top),
    viewportWidth: viewportRect.width,
    viewportHeight: viewportRect.height,
    padX: ISSUE_PAD_X,
  }
}

export function measureTrapLayout(
  viewport: HTMLElement,
  camera: HTMLElement,
  command: HTMLElement,
  options?: { neutralize?: boolean }
): TrapLayout | null {
  if (options?.neutralize !== false) {
    neutralizeCameraForMeasure(camera)
  }

  return readTrapLayoutAtRest(viewport, camera, command)
}

export function computeCameraMetrics(layout: TrapLayout): CameraMetrics {
  const panY = layout.focusCenterY - layout.cmdCenterY

  const zoomScale = Math.min(
    Math.max((layout.viewportHeight * 0.22) / layout.cmdHeight, 2.1),
    3.8
  )

  const zoomX = layout.padX - zoomScale * layout.cmdLeft
  const zoomY = layout.focusCenterY - zoomScale * layout.cmdCenterY
  const scanEndX =
    layout.viewportWidth - layout.padX - zoomScale * layout.cmdRight

  const scanDistance = Math.max(Math.abs(scanEndX - zoomX), 0)
  const scanMs = Math.max((scanDistance / SCAN_PX_PER_SEC) * 1000, MIN_SCAN_MS)

  const durationMs =
    OVERVIEW_MS +
    PAN_MS +
    HOLD_MS +
    ZOOM_MS +
    POST_ZOOM_WAIT_MS +
    scanMs +
    TAIL_MS

  const overviewEnd = OVERVIEW_MS / durationMs
  const panEnd = (OVERVIEW_MS + PAN_MS) / durationMs
  const holdEnd = (OVERVIEW_MS + PAN_MS + HOLD_MS) / durationMs
  const zoomEnd = (OVERVIEW_MS + PAN_MS + HOLD_MS + ZOOM_MS) / durationMs
  const postZoomEnd =
    (OVERVIEW_MS + PAN_MS + HOLD_MS + ZOOM_MS + POST_ZOOM_WAIT_MS) /
    durationMs
  const scanEndPhase =
    (OVERVIEW_MS +
      PAN_MS +
      HOLD_MS +
      ZOOM_MS +
      POST_ZOOM_WAIT_MS +
      scanMs) /
    durationMs

  return {
    durationMs,
    panY,
    zoomX,
    zoomY,
    scanEndX,
    zoomScale,
    panEnd,
    holdEnd,
    zoomEnd,
    postZoomEnd,
    scanEnd: scanEndPhase,
    overviewEnd,
    annotationShowAfterMs: getAnnotationShowAfterMs(),
  }
}

export function measureCameraMetrics(
  viewport: HTMLElement,
  camera: HTMLElement,
  command: HTMLElement,
  options?: { neutralize?: boolean }
): CameraMetrics | null {
  const layout = measureTrapLayout(viewport, camera, command, options)

  if (!layout) {
    return null
  }

  return computeCameraMetrics(layout)
}

export function buildCameraKeyframes(metrics: CameraMetrics): Keyframe[] {
  const {
    panY,
    zoomX,
    zoomY,
    scanEndX,
    zoomScale,
    overviewEnd,
    panEnd,
    holdEnd,
    zoomEnd,
    postZoomEnd,
    scanEnd,
  } = metrics

  const zoomed = `translate3d(${zoomX}px, ${zoomY}px, 0) scale(${zoomScale})`

  return [
    { transform: "translate3d(0, 0, 0) scale(1)", offset: 0 },
    {
      transform: "translate3d(0, 0, 0) scale(1)",
      offset: overviewEnd,
    },
    {
      transform: `translate3d(0, ${panY}px, 0) scale(1)`,
      offset: panEnd,
      easing: PAN_SCROLL_EASING,
    },
    {
      transform: `translate3d(0, ${panY}px, 0) scale(1)`,
      offset: holdEnd,
      easing: "linear",
    },
    {
      transform: zoomed,
      offset: zoomEnd,
      easing: "ease-out",
    },
    {
      transform: zoomed,
      offset: postZoomEnd,
      easing: "linear",
    },
    {
      transform: `translate3d(${scanEndX}px, ${zoomY}px, 0) scale(${zoomScale})`,
      offset: scanEnd,
      easing: "linear",
    },
    {
      transform: `translate3d(${scanEndX}px, ${zoomY}px, 0) scale(${zoomScale})`,
      offset: 1,
    },
  ]
}

export function applyMetricsToAnimation(
  animation: Animation,
  metrics: CameraMetrics
): void {
  const effect = animation.effect

  if (!(effect instanceof KeyframeEffect)) {
    return
  }

  effect.setKeyframes(buildCameraKeyframes(metrics))
  effect.updateTiming({
    duration: metrics.durationMs,
    iterations: 1,
    easing: "linear",
    fill: "forwards",
  })
}

export function createCameraAnimation(
  camera: HTMLElement,
  metrics: CameraMetrics
): Animation {
  camera.getAnimations().forEach((animation) => {
    animation.cancel()
  })

  camera.style.transformOrigin = "0 0"

  return camera.animate(buildCameraKeyframes(metrics), {
    duration: metrics.durationMs,
    iterations: 1,
    easing: "linear",
    fill: "forwards",
  })
}

export function shouldShowTrapAnnotation(
  timeMs: number,
  showAfterMs: number
): boolean {
  return timeMs >= showAfterMs
}
