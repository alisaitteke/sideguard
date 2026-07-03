/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Static manifest for press assets served from /media-kit/* after build sync.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */

export const MEDIA_KIT_ZIP_URL = "/media-kit.zip"

export type MediaKitCategory = "logo" | "icon" | "banner" | "gallery"

export type MediaKitAsset = {
  path: string
  label: string
  category: MediaKitCategory
  dimensions?: string
  format: string
  previewBg?: "dark" | "light" | "checkered"
}

export const BRAND_COLORS = [
  { token: "Logo (light mode)", hex: "#0d9488", usage: "Light backgrounds, favicon" },
  { token: "Logo (dark mode)", hex: "#5eead4", usage: "Dark hero, gallery slides" },
  { token: "Hero background", hex: "#10161c", usage: "Banners, thumbnails" },
  { token: "Primary green", hex: "#3a9e6e", usage: "Eyebrow, accents" },
  { token: "Foreground", hex: "#e6edf3", usage: "Headlines on dark" },
  { token: "Muted text", hex: "#94a3b8", usage: "Body copy on dark" },
] as const

export const MEDIA_KIT_ASSETS: MediaKitAsset[] = [
  {
    path: "/media-kit/logos/logo-teal-dark.svg",
    label: "Logo — teal (dark UI)",
    category: "logo",
    format: "SVG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/logos/logo-teal-light.svg",
    label: "Logo — teal (light UI)",
    category: "logo",
    format: "SVG",
    previewBg: "light",
  },
  {
    path: "/media-kit/logos/logo-white.svg",
    label: "Logo — white",
    category: "logo",
    format: "SVG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/logos/logo-black.svg",
    label: "Logo — black",
    category: "logo",
    format: "SVG",
    previewBg: "light",
  },
  {
    path: "/media-kit/logos/logo-on-dark-512.png",
    label: "Logo on dark",
    category: "logo",
    dimensions: "512×512",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/logos/logo-on-dark-1024.png",
    label: "Logo on dark (large)",
    category: "logo",
    dimensions: "1024×1024",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/logos/logo-teal-light-512.png",
    label: "Logo teal — light UI",
    category: "logo",
    dimensions: "512×512",
    format: "PNG",
    previewBg: "light",
  },
  {
    path: "/media-kit/logos/logo-teal-dark-512.png",
    label: "Logo teal — dark UI",
    category: "logo",
    dimensions: "512×512",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/logos/thumbnail-240.png",
    label: "Product Hunt thumbnail",
    category: "logo",
    dimensions: "240×240",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/icons/app-icon-180.png",
    label: "App icon",
    category: "icon",
    dimensions: "180×180",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/icons/app-icon-512.png",
    label: "App icon (large)",
    category: "icon",
    dimensions: "512×512",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/banners/og-1200x630.png",
    label: "Open Graph",
    category: "banner",
    dimensions: "1200×630",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/banners/og-card-site.png",
    label: "OG card (site)",
    category: "banner",
    dimensions: "1200×630",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/banners/twitter-1600x900.png",
    label: "Twitter / X announcement",
    category: "banner",
    dimensions: "1600×900",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/banners/linkedin-1200x627.png",
    label: "LinkedIn post",
    category: "banner",
    dimensions: "1200×627",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/banners/social-1280x640.png",
    label: "Social header",
    category: "banner",
    dimensions: "1280×640",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/gallery/01-hero-1270x760.png",
    label: "Gallery — hero",
    category: "gallery",
    dimensions: "1270×760",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/gallery/02-approval-1270x760.png",
    label: "Gallery — approval flow",
    category: "gallery",
    dimensions: "1270×760",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/gallery/03-integrations-1270x760.png",
    label: "Gallery — integrations",
    category: "gallery",
    dimensions: "1270×760",
    format: "PNG",
    previewBg: "dark",
  },
  {
    path: "/media-kit/gallery/04-install-1270x760.png",
    label: "Gallery — install",
    category: "gallery",
    dimensions: "1270×760",
    format: "PNG",
    previewBg: "dark",
  },
]

export const GALLERY_VIDEO = {
  mp4: "/media-kit/gallery/05-prompt-injection-1270x760.mp4",
  poster: "/media-kit/gallery/01-hero-1270x760.png",
  label: "Prompt injection demo",
  dimensions: "1270×760",
}

export const LAUNCH_COPY_BLOCKS = [
  {
    id: "ph-tagline",
    title: "Product Hunt tagline (≤60 chars)",
    text: "MCP guard with human approval for Cursor & Claude Code",
  },
  {
    id: "ph-description",
    title: "Product Hunt description",
    text: `SideGuard is a local MCP guard for Cursor and Claude Code.

When vibe coding with AI agents, destructive shell commands and MCP tool calls can slip through fast. SideGuard intercepts them at the hook layer, applies your YAML policy, and holds risky actions for human approval—via the terminal CLI, optional macOS menu-bar tray, or alert-only notifications.

What you get:
• Human-in-the-loop approval for shell commands and MCP tools
• YAML policy you control (deny secret paths, allowlists, etc.)
• Fail-closed hooks—blocked by default when SideGuard is unavailable
• Local audit trail on your machine (no cloud proxy)
• Open source, macOS and Linux

https://sideguard.io`,
  },
  {
    id: "twitter",
    title: "Twitter / X launch tweet",
    text: `SideGuard is live — an MCP guard for Cursor & Claude Code.

Your AI wants to run a command? You approve it first. YAML policy, fail-closed hooks, local audit trail. No cloud proxy.

curl -fsSL https://sideguard.io/setup.sh | sh

https://sideguard.io`,
  },
  {
    id: "linkedin",
    title: "LinkedIn post opener",
    text: `Shipping SideGuard — a vibe coding security tool for developers using Cursor and Claude Code.

AI agents are fast. That's the point—and the risk. SideGuard adds human-in-the-loop approval before shell commands and MCP tools execute on your machine.

• YAML policy you own
• Fail-closed hooks
• Local audit trail
• No cloud proxy

Open source: https://github.com/alisaitteke/sideguard
Try it: https://sideguard.io`,
  },
  {
    id: "hn",
    title: "Show HN title + body",
    text: `Title: Show HN: SideGuard – MCP guard with human approval for Cursor and Claude Code

SideGuard intercepts shell commands and MCP tool calls from Cursor and Claude Code before they run. You approve or deny in the terminal (or macOS tray). Policy is YAML on disk; audit log stays local.

Open source (Go): https://github.com/alisaitteke/sideguard
Site: https://sideguard.io`,
  },
] as const

export const BRAND_GUIDELINES = [
  "Prefer shield + checkmark mark with teal/mint fill on dark backgrounds.",
  "Do not stretch, rotate, or add effects beyond the subtle glow used in kit assets.",
  "Minimum clear space: height of the checkmark circle on all sides.",
  'Spell the product name as "SideGuard" — one word, capital S and G.',
] as const

export const BOILERPLATE =
  "Vibe coding security tool — MCP guard with human-in-the-loop approval for Cursor and Claude Code."

export const META_BOILERPLATE =
  "SideGuard is a vibe coding security tool and MCP guard for Cursor and Claude Code. Human-in-the-loop approval for shell commands and MCP tools—not an MCP antivirus. YAML policy, fail-closed hooks, local audit trail."
