/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Browser router shell: home, media kit, contact, and catch-all 404.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import { BrowserRouter, Route, Routes } from "react-router-dom"

import { PostHogPageview } from "@/components/PostHogPageview"
import { PageShell } from "@/components/layout/PageShell"
import { HomePage } from "@/pages/HomePage"
import { ContactPage } from "@/pages/ContactPage"
import { MediaPage } from "@/pages/MediaPage"
import { NotFoundPage } from "@/pages/NotFoundPage"
import { PromptInjectionExportPage } from "@/pages/PromptInjectionExportPage"

export function AppRouter() {
  return (
    <BrowserRouter>
      <PostHogPageview />
      <Routes>
        <Route
          path="/export/prompt-injection"
          element={<PromptInjectionExportPage />}
        />
        <Route
          path="/export/prompt-injection-pan"
          element={<PromptInjectionExportPage clip="pan" />}
        />
        <Route element={<PageShell />}>
          <Route index element={<HomePage />} />
          <Route path="media" element={<MediaPage />} />
          <Route path="contact" element={<ContactPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
