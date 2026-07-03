/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Router layout shell: header, page outlet, and shared footer.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import { Outlet } from "react-router-dom"

import { SiteFooter } from "@/components/layout/SiteFooter"
import { SiteHeader } from "@/components/layout/SiteHeader"

export function PageShell() {
  return (
    <div className="flex min-h-svh flex-col bg-background text-foreground">
      <SiteHeader />
      <Outlet />
      <SiteFooter />
    </div>
  )
}
