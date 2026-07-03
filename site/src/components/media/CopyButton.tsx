/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Copy-to-clipboard control for press copy blocks on the media page.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import { useEffect, useRef, useState } from "react"
import { CheckIcon, CopyIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const COPIED_RESET_MS = 2000

type CopyButtonProps = {
  text: string
  label?: string
  className?: string
}

export function CopyButton({ text, label = "Copy", className }: CopyButtonProps) {
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
      await navigator.clipboard.writeText(text)
      setCopied(true)
      if (resetTimerRef.current) {
        clearTimeout(resetTimerRef.current)
      }
      resetTimerRef.current = setTimeout(() => setCopied(false), COPIED_RESET_MS)
    } catch {
      // Clipboard unavailable.
    }
  }

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      className={cn("shrink-0", className)}
      onClick={() => void handleCopy()}
      aria-label={copied ? "Copied" : label}
    >
      {copied ? <CheckIcon /> : <CopyIcon />}
      {copied ? "Copied" : label}
    </Button>
  )
}
