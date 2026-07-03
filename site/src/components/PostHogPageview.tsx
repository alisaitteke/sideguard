/**
 * Captures PostHog $pageview on client-side route changes (React Router SPA).
 */
import { useEffect } from "react"
import { useLocation } from "react-router-dom"

import { isPostHogEnabled, posthog } from "@/lib/posthog"

export function PostHogPageview() {
  const location = useLocation()

  useEffect(() => {
    if (!isPostHogEnabled) return
    posthog.capture("$pageview")
  }, [location.pathname, location.search, location.hash])

  return null
}
