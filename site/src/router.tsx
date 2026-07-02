/**
 * Browser router shell — home route plus catch-all 404.
 * Future routes: /docs/*, /changelog, /blog (not implemented in vss Phase 2).
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
import { BrowserRouter, Route, Routes } from "react-router-dom"

import { PostHogPageview } from "@/components/PostHogPageview"
import { PageShell } from "@/components/layout/PageShell"
import { HomePage } from "@/pages/HomePage"
import { NotFoundPage } from "@/pages/NotFoundPage"

export function AppRouter() {
  return (
    <BrowserRouter>
      <PostHogPageview />
      <Routes>
        <Route element={<PageShell />}>
          <Route index element={<HomePage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
