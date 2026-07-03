/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * PostHog analytics for the public landing site (pageviews + future product events).
 * Proxied via a.alisait.com; UI links resolve to eu.posthog.com.
 */
import posthog from "posthog-js"

const POSTHOG_KEY = "phc_DmZJs26RKAHkKkCxENCARveNwLgpMu9todLwpAnm5Fo4"

/** PostHog runs only in production builds (`vite build`); disabled during `vite dev`. */
export const isPostHogEnabled = import.meta.env.PROD

let initialized = false

/** Initialize PostHog once in the browser. Safe to call multiple times. */
export function initPostHog(): void {
  if (!isPostHogEnabled || initialized || typeof window === "undefined") return

  posthog.init(POSTHOG_KEY, {
    api_host: "https://a.alisait.com",
    ui_host: "https://eu.posthog.com",
    defaults: "2026-05-30",
    person_profiles: "always",
    capture_pageview: false,
  })

  initialized = true
}

export { posthog }
