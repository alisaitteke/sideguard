/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Sets per-route document title, meta description, and canonical URL for SPA pages.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import { useEffect } from "react"

import { SITE_URL } from "@/lib/seo"

function upsertMeta(name: string, content: string, attribute: "name" | "property" = "name") {
  let el = document.querySelector(`meta[${attribute}="${name}"]`)
  if (!el) {
    el = document.createElement("meta")
    el.setAttribute(attribute, name)
    document.head.appendChild(el)
  }
  el.setAttribute("content", content)
}

function upsertCanonical(href: string) {
  let el = document.querySelector('link[rel="canonical"]')
  if (!el) {
    el = document.createElement("link")
    el.setAttribute("rel", "canonical")
    document.head.appendChild(el)
  }
  el.setAttribute("href", href)
}

export function usePageMeta(title: string, description: string, path: string) {
  useEffect(() => {
    const previousTitle = document.title
    document.title = title
    upsertMeta("description", description)
    upsertMeta("og:title", title, "property")
    upsertMeta("og:description", description, "property")
    upsertMeta("twitter:title", title)
    upsertMeta("twitter:description", description)
    upsertCanonical(`${SITE_URL}${path}`)

    return () => {
      document.title = previousTitle
    }
  }, [title, description, path])
}
