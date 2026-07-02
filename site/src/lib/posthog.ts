/**
 * PostHog analytics for the public landing site (pageviews + future product events).
 * Proxied via kim.sideguard.io; UI links resolve to eu.posthog.com.
 */
import posthog from "posthog-js"

const POSTHOG_KEY = "phc_DmZJs26RKAHkKkCxENCARveNwLgpMu9todLwpAnm5Fo4"

let initialized = false

/** Initialize PostHog once in the browser. Safe to call multiple times. */
export function initPostHog(): void {
  if (initialized || typeof window === "undefined") return

  posthog.init(POSTHOG_KEY, {
    api_host: "https://kim.sideguard.io",
    ui_host: "https://eu.posthog.com",
    defaults: "2026-05-30",
    person_profiles: "always",
    capture_pageview: false,
  })

  initialized = true
}

export { posthog }
