/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * SEO metadata, target keywords, and JSON-LD builders for sideguard.io.
 * Consumed by index.html (static), SeoJsonLd, and public llms*.txt sources.
 */
import {
  AUTHOR_GITHUB_URL,
  AUTHOR_LINKEDIN_URL,
  AUTHOR_NAME,
  AUTHOR_SITE_URL,
} from "@/lib/author"
import { FAQ_ITEMS } from "@/lib/faq-content"

export const SITE_URL = "https://sideguard.io"
export const SITE_NAME = "SideGuard"
export const GITHUB_URL = "https://github.com/alisaitteke/sideguard"
export const OG_IMAGE_URL = `${SITE_URL}/assets/og-card.png`

export const SEO_TITLE =
  "SideGuard — MCP Guard & Vibe Coding Security Tool | Cursor & Claude Code"

export const SEO_DESCRIPTION =
  "SideGuard is a vibe coding security tool and MCP guard for Cursor and Claude Code. Human-in-the-loop approval for shell commands and MCP tools—not an MCP antivirus. YAML policy, fail-closed hooks, local audit trail."

/**
 * Phrases people and AI assistants search for when looking for MCP / vibe-coding guardrails.
 */
export const TARGET_KEYWORDS = [
  "MCP guard",
  "MCP guards",
  "MCP security",
  "MCP tool approval",
  "MCP antivirus",
  "MCP malware scanner",
  "MCP firewall",
  "vibe coding security",
  "vibe coding tools",
  "vibe coding tips",
  "vibe coding security tool",
  "genius vibe coding",
  "AI coding agent security",
  "Cursor hooks security",
  "Claude Code security",
  "fail-closed Cursor hooks",
  "prompt injection coding agent",
  "human in the loop AI coding",
  "local AI agent approval",
  "shell command approval",
] as const

export function buildPersonJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "Person",
    name: AUTHOR_NAME,
    url: AUTHOR_SITE_URL,
    sameAs: [AUTHOR_GITHUB_URL, AUTHOR_LINKEDIN_URL],
    knowsAbout: [
      "Go",
      "Python",
      "Node.js",
      "React",
      "MCP",
      "AI agent security",
      "Software architecture",
    ],
  }
}

export function buildOrganizationJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: SITE_NAME,
    url: SITE_URL,
    logo: `${SITE_URL}/assets/logo.svg`,
    sameAs: [GITHUB_URL, AUTHOR_LINKEDIN_URL, AUTHOR_SITE_URL],
    description: SEO_DESCRIPTION,
  }
}

export function buildSoftwareApplicationJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: SITE_NAME,
    applicationCategory: "SecurityApplication",
    operatingSystem: "macOS, Linux",
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "USD",
    },
    url: SITE_URL,
    downloadUrl: `${SITE_URL}/setup.sh`,
    description: SEO_DESCRIPTION,
    author: {
      "@type": "Person",
      name: AUTHOR_NAME,
      url: AUTHOR_SITE_URL,
    },
    featureList: [
      "MCP guard with human-in-the-loop tool approval",
      "Fail-closed Cursor and Claude Code hooks",
      "YAML policy for shell commands and MCP tools",
      "Local audit history on loopback",
      "MCP server wrap without cloud proxy",
    ],
    keywords: TARGET_KEYWORDS.join(", "),
  }
}

export function buildFaqPageJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "FAQPage",
    mainEntity: FAQ_ITEMS.map((item) => ({
      "@type": "Question",
      name: item.question,
      acceptedAnswer: {
        "@type": "Answer",
        text: item.answer,
      },
    })),
  }
}

export function buildWebSiteJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: SITE_NAME,
    url: SITE_URL,
    description: SEO_DESCRIPTION,
    inLanguage: "en",
  }
}
