/**
 * YouTube-style chrome for the prompt-injection WAAPI timeline.
 * Control layout mirrors the Web Dev Simplified player clone pattern.
 */
import { useRef, type CSSProperties } from "react"
import { PauseIcon, PlayIcon } from "lucide-react"

import type { PromptInjectionPlayback } from "@/components/sections/use-prompt-injection-zoom"
import { cn } from "@/lib/utils"

import "@/styles/prompt-injection-yt-player.css"

function formatTime(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60

  return `${minutes}:${seconds.toString().padStart(2, "0")}`
}

type PromptInjectionPlayerProps = {
  playback: PromptInjectionPlayback
}

/** Center play + dim overlay while paused or before first play. */
export function PromptInjectionCenterPlay({
  playback,
}: PromptInjectionPlayerProps) {
  const { ready, isPlaying, isScrubbing, play } = playback

  if (!ready || isPlaying || isScrubbing) {
    return null
  }

  return (
    <div
      className="yt-paused-overlay"
      onClick={play}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault()
          play()
        }
      }}
      role="button"
      tabIndex={0}
      aria-label="Play animation"
    >
      <div className="yt-paused-center">
        <span className="yt-big-play" aria-hidden>
          <PlayIcon className="size-7 fill-white text-white" />
        </span>
        <p className="yt-paused-caption">Uh… what is this?</p>
      </div>
    </div>
  )
}

export function PromptInjectionPlayer({ playback }: PromptInjectionPlayerProps) {
  const resumeAfterScrubRef = useRef(false)

  const {
    ready,
    durationMs,
    currentTimeMs,
    isPlaying,
    isScrubbing,
    togglePlayPause,
    seek,
    beginScrub,
    endScrub,
  } = playback

  if (!ready || durationMs <= 0) {
    return null
  }

  const progress = Math.min(currentTimeMs / durationMs, 1)
  const timelineStyle = {
    "--progress-position": progress,
  } as CSSProperties

  return (
    <div
      className={cn(
        "yt-video-controls-container",
        !isPlaying && "paused",
        isScrubbing && "scrubbing"
      )}
      role="group"
      aria-label="Animation playback controls"
    >
      <div className="yt-timeline-container">
        <div className="yt-timeline" style={timelineStyle}>
          <div className="yt-thumb-indicator" aria-hidden />
        </div>
        <input
          type="range"
          className="yt-timeline-input"
          min={0}
          max={durationMs}
          step={50}
          value={Math.min(currentTimeMs, durationMs)}
          aria-label="Seek animation"
          aria-valuemin={0}
          aria-valuemax={durationMs}
          aria-valuenow={currentTimeMs}
          aria-valuetext={`${formatTime(currentTimeMs)} of ${formatTime(durationMs)}`}
          onPointerDown={() => {
            resumeAfterScrubRef.current = isPlaying
            beginScrub()
          }}
          onChange={(event) => {
            seek(Number(event.target.value))
          }}
          onPointerUp={() => {
            endScrub(resumeAfterScrubRef.current)
          }}
          onKeyUp={(event) => {
            if (event.key === "Enter" || event.key === " ") {
              endScrub(resumeAfterScrubRef.current)
            }
          }}
        />
      </div>

      <div className="yt-controls">
        <button
          type="button"
          className="yt-control-btn"
          onClick={togglePlayPause}
          aria-label={isPlaying ? "Pause" : "Play"}
        >
          {isPlaying ? (
            <PauseIcon aria-hidden className="size-[1.1rem]" />
          ) : (
            <PlayIcon aria-hidden className="size-[1.1rem]" />
          )}
        </button>

        <div className="yt-duration-container">
          <span className="tabular-nums">{formatTime(currentTimeMs)}</span>
          <span className="yt-duration-sep">/</span>
          <span className="tabular-nums">{formatTime(durationMs)}</span>
        </div>
      </div>
    </div>
  )
}
