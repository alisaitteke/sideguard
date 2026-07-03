/**
 * Copies repo-root media-kit into site/dist for GitHub Pages static serving.
 * Optionally creates dist/media-kit.zip for bulk download from /media page.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import { cpSync, existsSync, mkdirSync } from "node:fs"
import { execSync } from "node:child_process"
import path from "node:path"
import { fileURLToPath } from "node:url"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const siteRoot = path.resolve(__dirname, "..")
const repoRoot = path.resolve(siteRoot, "..")
const mediaKitSrc = path.join(repoRoot, "media-kit")
const distDir = path.join(siteRoot, "dist")
const publicDir = path.join(siteRoot, "public")
const mediaKitDest = path.join(distDir, "media-kit")
const publicMediaKitDest = path.join(publicDir, "media-kit")
const zipDest = path.join(distDir, "media-kit.zip")
const publicZipDest = path.join(publicDir, "media-kit.zip")

if (!existsSync(mediaKitSrc)) {
  console.error(`media-kit not found at ${mediaKitSrc}`)
  process.exit(1)
}

if (!existsSync(distDir)) {
  mkdirSync(distDir, { recursive: true })
}

cpSync(mediaKitSrc, mediaKitDest, { recursive: true })
console.log(`Synced ${mediaKitSrc} → ${mediaKitDest}`)

cpSync(mediaKitSrc, publicMediaKitDest, { recursive: true })
console.log(`Synced ${mediaKitSrc} → ${publicMediaKitDest}`)

try {
  execSync(`zip -rq "${zipDest}" media-kit`, { cwd: distDir, stdio: "inherit" })
  console.log(`Created ${zipDest}`)
  cpSync(zipDest, publicZipDest)
  console.log(`Synced zip → ${publicZipDest}`)
} catch {
  console.warn("zip not available — skipping media-kit.zip (install zip on CI if needed)")
}
