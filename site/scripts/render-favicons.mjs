/**
 * Renders favicon PNG fallbacks from brand tokens + logo.svg path.
 *
 * Outputs:
 * - site/public/assets/favicon-16.png
 * - site/public/assets/favicon-32.png
 * - site/public/assets/apple-touch-icon.png (180×180)
 *
 * Usage: npm run render:favicons  (site/)
 */
import { writeFileSync, mkdirSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"
import { chromium } from "playwright"

import { BRAND, readLogoPath } from "./brand-tokens.mjs"

const __dirname = dirname(fileURLToPath(import.meta.url))
const assetsDir = join(__dirname, "../public/assets")

const logoPath = readLogoPath()

/**
 * @param {number} size
 * @param {string} fill
 * @param {string} background
 */
function buildIconHtml(size, fill, background) {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      width: ${size}px;
      height: ${size}px;
      overflow: hidden;
      background: ${background};
    }
    svg {
      width: 100%;
      height: 100%;
      display: block;
    }
  </style>
</head>
<body>
  <svg viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
    <path fill="${fill}" d="${logoPath}" />
  </svg>
</body>
</html>`
}

const OUTPUTS = [
  {
    name: "favicon-16.png",
    size: 16,
    fill: BRAND.logoLight,
    background: "transparent",
  },
  {
    name: "favicon-32.png",
    size: 32,
    fill: BRAND.logoLight,
    background: "transparent",
  },
  {
    name: "favicon-16-dark.png",
    size: 16,
    fill: BRAND.logoDark,
    background: "transparent",
  },
  {
    name: "favicon-32-dark.png",
    size: 32,
    fill: BRAND.logoDark,
    background: "transparent",
  },
  {
    name: "apple-touch-icon.png",
    size: 180,
    fill: BRAND.logoDark,
    background: BRAND.heroBackground,
  },
]

async function render() {
  mkdirSync(assetsDir, { recursive: true })
  const browser = await chromium.launch()
  const page = await browser.newPage()

  for (const { name, size, fill, background } of OUTPUTS) {
    const outPath = join(assetsDir, name)
    await page.setViewportSize({ width: size, height: size })
    await page.setContent(buildIconHtml(size, fill, background), {
      waitUntil: "networkidle",
    })
    await page.waitForTimeout(80)
    const buffer = await page.screenshot({
      type: "png",
      omitBackground: background === "transparent",
    })
    writeFileSync(outPath, buffer)
    console.log(`Wrote ${outPath} (${size}×${size})`)
  }

  await browser.close()
}

render().catch((err) => {
  console.error(err)
  process.exit(1)
})
