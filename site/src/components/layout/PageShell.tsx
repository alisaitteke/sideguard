/**
 * Router layout shell: page outlet and shared footer.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
import { Outlet } from "react-router-dom"

import { SiteFooter } from "@/components/layout/SiteFooter"

export function PageShell() {
  return (
    <div className="flex min-h-svh flex-col bg-background text-foreground">
      <Outlet />
      <SiteFooter />
    </div>
  )
}
