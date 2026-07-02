/**
 * Vite config for SideGuard GitHub Pages SPA (apex domain sideguard.io).
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-1.0-scaffold.md
 */
import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

/**
 * Vite config for SideGuard GitHub Pages SPA (apex domain sideguard.io).
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-1.0-scaffold.md
 */
export default defineConfig({
  base: "/",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
})
