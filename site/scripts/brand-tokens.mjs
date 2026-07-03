/**
 * Shared SideGuard brand tokens for static asset render scripts.
 * Keep in sync with --hero-logo in site/src/index.css.
 */
import { readFileSync } from "node:fs"
import { join, dirname } from "node:path"
import { fileURLToPath } from "node:url"

const __dirname = dirname(fileURLToPath(import.meta.url))
const siteRoot = join(__dirname, "..")

export const BRAND = {
  logoLight: "#0d9488",
  logoDark: "#5eead4",
  heroBackground: "#10161c",
  glowMint: "rgba(94, 234, 212, 0.22)",
}

/** Shield + checkmark path from public logo.svg (mask-friendly source of truth). */
export function readLogoPath() {
  const logoSvg = readFileSync(join(siteRoot, "public/assets/logo.svg"), "utf8")
  const match = logoSvg.match(/<path[^>]+d="([^"]+)"/)
  if (!match) {
    throw new Error("Could not extract logo path from public/assets/logo.svg")
  }
  return match[1]
}
