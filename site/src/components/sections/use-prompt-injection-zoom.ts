import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type RefObject,
} from "react"

import {
  createCameraAnimation,
  measureCameraMetrics,
  type CameraMetrics,
} from "@/components/sections/prompt-injection-camera-metrics"
import {
  createTrapLineAnnotation,
  TrapAnnotationController,
} from "@/components/sections/prompt-injection-trap-annotation"

export type { CameraMetrics } from "@/components/sections/prompt-injection-camera-metrics"

export type PromptInjectionPlayback = {
  ready: boolean
  durationMs: number
  currentTimeMs: number
  isPlaying: boolean
  isScrubbing: boolean
  play: () => void
  pause: () => void
  togglePlayPause: () => void
  seek: (timeMs: number) => void
  beginScrub: () => void
  endScrub: (resume: boolean) => void
}

function clampTime(timeMs: number, durationMs: number): number {
  return Math.min(Math.max(timeMs, 0), durationMs)
}

type SceneRuntime = {
  metrics: CameraMetrics
  animation: Animation
  annotation: TrapAnnotationController
  loopHandler: () => void
}

/**
 * Drives the GitHub-issue camera inside the content pane (below fixed browser chrome).
 * Layout + zoom + annotation share `prompt-injection-camera-metrics` and recompute on resize.
 */
export function usePromptInjectionZoom(
  shellRef: RefObject<HTMLElement | null>,
  viewportRef: RefObject<HTMLElement | null>,
  cameraRef: RefObject<HTMLElement | null>,
  commandRef: RefObject<HTMLElement | null>,
  onReady: (ready: boolean) => void
): PromptInjectionPlayback {
  const runtimeRef = useRef<SceneRuntime | null>(null)
  const durationMsRef = useRef(0)
  const playingRef = useRef(false)
  const scrubbingRef = useRef(false)

  const [ready, setReady] = useState(false)
  const [durationMs, setDurationMs] = useState(0)
  const [currentTimeMs, setCurrentTimeMs] = useState(0)
  const [isPlaying, setIsPlaying] = useState(false)
  const [isScrubbing, setIsScrubbing] = useState(false)

  const syncSceneAtTime = useCallback((timeMs: number) => {
    const runtime = runtimeRef.current

    if (!runtime) {
      return
    }

    const clamped = clampTime(timeMs, runtime.metrics.durationMs)
    runtime.annotation.sync(clamped, runtime.metrics.annotationShowAfterMs)
    setCurrentTimeMs(clamped)
  }, [])

  const syncTimeFromAnimation = useCallback(() => {
    const runtime = runtimeRef.current

    if (!runtime) {
      return
    }

    syncSceneAtTime(runtime.animation.currentTime ?? 0)
  }, [syncSceneAtTime])

  const play = useCallback(() => {
    const runtime = runtimeRef.current

    if (!runtime) {
      return
    }

    const { animation, metrics } = runtime
    const atEnd =
      (animation.currentTime ?? 0) >= metrics.durationMs - 16

    if (atEnd) {
      animation.currentTime = 0
      syncSceneAtTime(0)
    }

    playingRef.current = true
    animation.play()
    setIsPlaying(true)
  }, [syncSceneAtTime])

  const pause = useCallback(() => {
    const runtime = runtimeRef.current

    if (!runtime) {
      return
    }

    playingRef.current = false
    runtime.animation.pause()
    setIsPlaying(false)
    syncTimeFromAnimation()
  }, [syncTimeFromAnimation])

  const togglePlayPause = useCallback(() => {
    if (playingRef.current) {
      pause()
    } else {
      play()
    }
  }, [pause, play])

  const seek = useCallback(
    (timeMs: number) => {
      const runtime = runtimeRef.current

      if (!runtime) {
        return
      }

      const nextTime = clampTime(timeMs, runtime.metrics.durationMs)
      runtime.animation.currentTime = nextTime
      syncSceneAtTime(nextTime)
    },
    [syncSceneAtTime]
  )

  const beginScrub = useCallback(() => {
    scrubbingRef.current = true
    setIsScrubbing(true)
    pause()
  }, [pause])

  const endScrub = useCallback(
    (resume: boolean) => {
      scrubbingRef.current = false
      setIsScrubbing(false)
      syncTimeFromAnimation()

      if (resume) {
        play()
      }
    },
    [play, syncTimeFromAnimation]
  )

  useEffect(() => {
    if (!ready) {
      return undefined
    }

    let rafId = 0

    const tick = () => {
      const runtime = runtimeRef.current

      if (
        runtime &&
        runtime.animation.playState === "running" &&
        !scrubbingRef.current
      ) {
        syncSceneAtTime(runtime.animation.currentTime ?? 0)
      }

      rafId = requestAnimationFrame(tick)
    }

    rafId = requestAnimationFrame(tick)

    return () => {
      cancelAnimationFrame(rafId)
    }
  }, [ready, syncSceneAtTime])

  useLayoutEffect(() => {
    const shell = shellRef.current
    const viewport = viewportRef.current
    const camera = cameraRef.current
    const command = commandRef.current

    if (!shell || !viewport || !camera || !command) {
      return undefined
    }

    const prefersReducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)"
    ).matches

    let staticAnnotationCleanup: (() => void) | null = null

    const disposeRuntime = () => {
      const runtime = runtimeRef.current

      if (!runtime) {
        return
      }

      runtime.animation.removeEventListener("finish", runtime.loopHandler)
      runtime.animation.cancel()
      runtime.annotation.dispose()
      runtimeRef.current = null
    }

    const relayoutScene = () => {
      const viewportEl = viewportRef.current
      const cameraEl = cameraRef.current
      const commandEl = commandRef.current
      const runtime = runtimeRef.current

      if (!viewportEl || !cameraEl || !commandEl || !runtime) {
        return
      }

      const savedTime = clampTime(
        runtime.animation.currentTime ?? 0,
        runtime.metrics.durationMs
      )
      const wasPlaying = playingRef.current

      const metrics = measureCameraMetrics(viewportEl, cameraEl, commandEl)

      if (!metrics) {
        return
      }

      runtime.animation.removeEventListener("finish", runtime.loopHandler)
      runtime.animation.cancel()

      const animation = createCameraAnimation(cameraEl, metrics)
      animation.addEventListener("finish", runtime.loopHandler)

      runtime.metrics = metrics
      runtime.animation = animation
      durationMsRef.current = metrics.durationMs
      setDurationMs(metrics.durationMs)

      const restoredTime = clampTime(savedTime, metrics.durationMs)
      animation.currentTime = restoredTime
      animation.pause()

      runtime.annotation.relayout()
      syncSceneAtTime(restoredTime)

      if (wasPlaying) {
        animation.play()
        setIsPlaying(true)
      } else {
        setIsPlaying(false)
      }
    }

    const bootstrapScene = () => {
      const viewportEl = viewportRef.current
      const cameraEl = cameraRef.current
      const commandEl = commandRef.current

      if (!viewportEl || !cameraEl || !commandEl) {
        disposeRuntime()
        durationMsRef.current = 0
        setReady(false)
        setDurationMs(0)
        onReady(false)
        return
      }

      staticAnnotationCleanup?.()
      staticAnnotationCleanup = null

      if (prefersReducedMotion) {
        disposeRuntime()
        durationMsRef.current = 0
        playingRef.current = false
        setReady(false)
        setDurationMs(0)
        setIsPlaying(false)

        const metrics = measureCameraMetrics(viewportEl, cameraEl, commandEl)

        if (metrics) {
          cameraEl.style.transform = `translate3d(${metrics.scanEndX}px, ${metrics.zoomY}px, 0) scale(${metrics.zoomScale})`
          const staticAnnotation = createTrapLineAnnotation(commandEl, false)
          staticAnnotation.show()
          staticAnnotationCleanup = () => {
            staticAnnotation.remove()
          }
        }

        onReady(false)
        return
      }

      const previousRuntime = runtimeRef.current
      const previousDuration = durationMsRef.current
      const previousTime = previousRuntime?.animation.currentTime ?? 0
      const timeRatio =
        previousDuration > 0 ? previousTime / previousDuration : 0

      const metrics = measureCameraMetrics(viewportEl, cameraEl, commandEl)

      if (!metrics) {
        disposeRuntime()
        durationMsRef.current = 0
        setReady(false)
        setDurationMs(0)
        onReady(false)
        return
      }

      cameraEl.style.removeProperty("transform")

      if (previousRuntime) {
        previousRuntime.animation.removeEventListener(
          "finish",
          previousRuntime.loopHandler
        )
        previousRuntime.animation.cancel()
      }

      const animation = createCameraAnimation(cameraEl, metrics)

      const annotation =
        previousRuntime?.annotation ?? new TrapAnnotationController()
      annotation.bind(commandEl)

      const loopHandler = () => {
        const runtime = runtimeRef.current

        if (!playingRef.current || !runtime || runtime.animation !== animation) {
          return
        }

        animation!.currentTime = 0
        syncSceneAtTime(0)
        annotation.relayout()
        animation!.play()
      }

      animation.addEventListener("finish", loopHandler)

      runtimeRef.current = {
        metrics,
        animation,
        annotation,
        loopHandler,
      }

      durationMsRef.current = metrics.durationMs
      setDurationMs(metrics.durationMs)
      setReady(true)

      const restoredTime = clampTime(
        timeRatio * metrics.durationMs,
        metrics.durationMs
      )
      animation.currentTime = restoredTime
      syncSceneAtTime(restoredTime)
      annotation.relayout()

      if (playingRef.current) {
        animation.play()
        setIsPlaying(true)
      } else {
        animation.pause()
        setIsPlaying(false)
      }

      onReady(true)
    }

    bootstrapScene()

    const resizeObserver = new ResizeObserver(() => {
      if (runtimeRef.current) {
        relayoutScene()
      } else {
        bootstrapScene()
      }
    })

    resizeObserver.observe(shell)
    resizeObserver.observe(viewport)

    window.addEventListener("resize", relayoutScene)

    if (document.fonts?.ready) {
      void document.fonts.ready.then(relayoutScene)
    }

    const raf = window.requestAnimationFrame(() => {
      window.requestAnimationFrame(relayoutScene)
    })

    return () => {
      window.cancelAnimationFrame(raf)
      resizeObserver.disconnect()
      window.removeEventListener("resize", relayoutScene)
      disposeRuntime()
      staticAnnotationCleanup?.()
      onReady(false)
    }
  }, [
    shellRef,
    viewportRef,
    cameraRef,
    commandRef,
    onReady,
    syncSceneAtTime,
  ])

  return {
    ready,
    durationMs,
    currentTimeMs,
    isPlaying,
    isScrubbing,
    play,
    pause,
    togglePlayPause,
    seek,
    beginScrub,
    endScrub,
  }
}
