/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Shared top navigation for marketing pages: home, media kit, contact, and GitHub.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import { Link, NavLink } from "react-router-dom"

import { GITHUB_URL } from "@/lib/seo"
import { cn } from "@/lib/utils"

const NAV_ITEMS = [
  { to: "/", label: "Home", end: true },
  { to: "/media", label: "Media", end: false },
  { to: "/contact", label: "Contact", end: false },
] as const

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-50 border-b border-border bg-background/90 backdrop-blur-md">
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between gap-4 px-4">
        <Link
          to="/"
          className="flex items-center gap-2.5 text-foreground transition-opacity hover:opacity-80"
        >
          <span
            aria-hidden
            className="size-7 bg-primary mask-[url(/assets/logo.svg)] mask-contain mask-center mask-no-repeat"
          />
          <span className="text-sm font-semibold tracking-tight">SideGuard</span>
        </Link>

        <nav aria-label="Main" className="flex items-center gap-1">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-muted text-foreground"
                    : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                )
              }
            >
              {item.label}
            </NavLink>
          ))}
          <a
            href={GITHUB_URL}
            target="_blank"
            rel="noopener noreferrer"
            aria-label="SideGuard on GitHub"
            className="ml-1 inline-flex size-7 items-center justify-center rounded-[min(var(--radius-md),12px)] text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground"
          >
            <svg
              aria-hidden
              viewBox="0 0 24 24"
              className="size-4 fill-current"
            >
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z" />
            </svg>
          </a>
        </nav>
      </div>
    </header>
  )
}
