/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Inline nav links reused in header and footer.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import { Link } from "react-router-dom"

import { cn } from "@/lib/utils"

const LINKS = [
  { to: "/", label: "Home" },
  { to: "/media", label: "Media" },
  { to: "/contact", label: "Contact" },
] as const

type SiteNavProps = {
  className?: string
}

export function SiteNav({ className }: SiteNavProps) {
  return (
    <nav aria-label="Site" className={cn("flex flex-wrap justify-center gap-x-4 gap-y-1", className)}>
      {LINKS.map((link) => (
        <Link
          key={link.to}
          to={link.to}
          className="text-primary hover:underline"
        >
          {link.label}
        </Link>
      ))}
    </nav>
  )
}
