/**
 * Landing hero: value prop, tagline, and positioning for vibe coders.
 */
import { useEffect, useRef, useState } from "react"
import { CheckIcon, ChevronDownIcon, CopyIcon } from "lucide-react"

import { HeroBackgroundGlow } from "@/components/sections/HeroBackgroundGlow"
import { INSTALL_COMMAND } from "@/lib/install"
import { smoothScrollToElement } from "@/lib/pan-scroll-motion"

const COPIED_RESET_MS = 2000

export function HeroSection() {
  const [copied, setCopied] = useState(false)
  const resetTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (resetTimerRef.current) {
        clearTimeout(resetTimerRef.current)
      }
    }
  }, [])

  async function handleCopy() {
    try {
      if (!navigator.clipboard?.writeText) {
        return
      }

      await navigator.clipboard.writeText(INSTALL_COMMAND)
      setCopied(true)

      if (resetTimerRef.current) {
        clearTimeout(resetTimerRef.current)
      }

      resetTimerRef.current = setTimeout(() => {
        setCopied(false)
      }, COPIED_RESET_MS)
    } catch {
      // Clipboard unavailable — no UI feedback needed.
    }
  }

  return (
    <header className="relative flex min-h-svh w-full flex-col items-center justify-center overflow-hidden border-b border-border bg-hero px-4 py-8 pb-16 text-center">
      <HeroBackgroundGlow />
      <div className="relative z-10 mx-auto w-full max-w-[52rem]">
        <div className="mb-8 flex flex-col items-center gap-4">
          <span
            aria-hidden
            className="size-[88px] bg-hero-glow mask-[url(/assets/logo.svg)] mask-contain mask-center mask-no-repeat"
          />
          <h1 className="text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
            SideGuard
          </h1>
        </div>
        <p className="mx-auto mt-4 max-w-[32rem] text-base leading-relaxed text-muted-foreground sm:text-lg">
          Before your AI assistant runs a command on your computer, SideGuard asks
          you to approve. Nothing runs until you say yes or no.
        </p>
        <div className="mt-10 flex justify-center sm:mt-12">
          <button
            type="button"
            onClick={() => void handleCopy()}
            aria-label={
              copied
                ? "Install command copied"
                : `Copy install command: ${INSTALL_COMMAND}`
            }
            className="install-terminal inline-flex w-fit max-w-full cursor-pointer flex-col overflow-hidden rounded-lg text-left"
          >
            <div
              aria-hidden
              className="install-terminal-chrome flex items-center gap-1.5 px-3 py-2"
            >
              <span className="size-2 rounded-full bg-[#ff5f57]" />
              <span className="size-2 rounded-full bg-[#febc2e]" />
              <span className="size-2 rounded-full bg-[#28c840]" />
            </div>
            <div className="install-terminal-body flex items-center gap-1 overflow-x-auto px-3 py-2 font-mono text-sm">
              <span aria-hidden className="install-terminal-prompt select-none">
                ❯
              </span>
              <code className="whitespace-nowrap">{INSTALL_COMMAND}</code>
              <span
                aria-hidden
                className="install-terminal-copy inline-flex size-6 shrink-0 items-center justify-center transition-colors"
              >
                {copied ? (
                  <CheckIcon className="size-3.5 text-[#3a9e6e]" />
                ) : (
                  <CopyIcon className="size-3.5" />
                )}
              </span>
            </div>
          </button>
        </div>
      </div>
      <a
        href="#prompt-injection"
        aria-label="Scroll down"
        className="absolute bottom-6 left-1/2 z-10 -translate-x-1/2 text-muted-foreground transition-colors hover:text-foreground"
        onClick={(event) => {
          const target = document.getElementById("prompt-injection")

          if (!target) {
            return
          }

          event.preventDefault()
          smoothScrollToElement(target)
        }}
      >
        <ChevronDownIcon className="size-6 animate-bounce" aria-hidden />
      </a>
    </header>
  )
}
