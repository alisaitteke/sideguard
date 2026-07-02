/**
 * Renders SideGuard brand PNGs from shared tokens (Geist, hero glow, teal logo).
 *
 * Outputs:
 * - site/public/assets/og-card.png (1200×630)
 * - .github/social-preview.png (1280×640)
 * - assets/readme-hero.png (1280×360) — GitHub README banner
 *
 * Usage: npm run render:social-card  (site/)
 */
import { readFileSync, writeFileSync, mkdirSync } from "node:fs"
import { dirname, join, resolve } from "node:path"
import { fileURLToPath } from "node:url"
import { chromium } from "playwright"

const __dirname = dirname(fileURLToPath(import.meta.url))
const siteRoot = resolve(__dirname, "..")
const repoRoot = resolve(siteRoot, "..")

const geistWoff2 = readFileSync(
  join(
    siteRoot,
    "node_modules/@fontsource-variable/geist/files/geist-latin-wght-normal.woff2",
  ),
)
const geistBase64 = geistWoff2.toString("base64")

const logoSvg = readFileSync(join(siteRoot, "public/assets/logo.svg"), "utf8")
const logoPath = logoSvg.match(/<path[^>]+d="([^"]+)"/)?.[1] ?? ""

/** @typedef {"social" | "readme"} CardVariant */

const OUTPUTS = [
  {
    variant: "social",
    width: 1200,
    height: 630,
    path: join(siteRoot, "public/assets/og-card.png"),
  },
  {
    variant: "social",
    width: 1280,
    height: 640,
    path: join(repoRoot, ".github/social-preview.png"),
  },
  {
    variant: "readme",
    width: 1280,
    height: 360,
    path: join(repoRoot, "assets/readme-hero.png"),
  },
]

/**
 * @param {number} width
 * @param {number} height
 * @param {CardVariant} variant
 */
function buildHtml(width, height, variant) {
  const scale = width / 1280
  const logoSize = Math.round((variant === "readme" ? 72 : 96) * scale)
  const titleSize = Math.round((variant === "readme" ? 48 : 58) * scale)
  const eyebrowSize = Math.round((variant === "readme" ? 12 : 13) * scale)
  const subtitleSize = Math.round((variant === "readme" ? 18 : 21) * scale)
  const domainSize = Math.round(14 * scale)
  const showDomain = variant === "social"
  const landingOrder = variant === "readme"

  const titleBlock = landingOrder
    ? `<h1 class="title">SideGuard</h1>
      <p class="eyebrow">Vibe Coding Security Tool</p>`
    : `<p class="eyebrow">Vibe Coding Security Tool</p>
      <h1 class="title">SideGuard</h1>
      <div class="rule" aria-hidden="true"></div>`

  const subtitle = landingOrder
    ? `<p class="subtitle">MCP guard with human-in-the-loop approval for Cursor and Claude Code</p>`
    : `<p class="subtitle">MCP guard for Cursor &amp; Claude Code</p>`

  const domainMarkup = showDomain
    ? `<span class="domain">sideguard.io</span>`
    : ""

  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <style>
    @font-face {
      font-family: "Geist";
      src: url(data:font/woff2;base64,${geistBase64}) format("woff2");
      font-weight: 100 900;
      font-style: normal;
    }

    * { margin: 0; padding: 0; box-sizing: border-box; }

    body {
      width: ${width}px;
      height: ${height}px;
      overflow: hidden;
      font-family: "Geist", system-ui, sans-serif;
      background: #10161c;
      color: #e8edf4;
      -webkit-font-smoothing: antialiased;
    }

    .canvas {
      position: relative;
      width: 100%;
      height: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .glow-primary {
      position: absolute;
      top: -18%;
      left: 50%;
      transform: translateX(-50%);
      width: 72%;
      height: 68%;
      background: radial-gradient(
        ellipse at center,
        rgba(35, 107, 71, 0.38) 0%,
        rgba(45, 134, 89, 0.14) 42%,
        transparent 72%
      );
      pointer-events: none;
    }

    .glow-accent {
      position: absolute;
      top: 8%;
      left: 50%;
      transform: translateX(-50%);
      width: 48%;
      height: 42%;
      background: radial-gradient(
        ellipse at center,
        rgba(94, 234, 212, 0.12) 0%,
        transparent 70%
      );
      pointer-events: none;
    }

    .frame {
      position: relative;
      z-index: 1;
      display: flex;
      flex-direction: column;
      align-items: center;
      text-align: center;
      padding: 0 ${Math.round(64 * scale)}px;
      max-width: 92%;
    }

    .logo-wrap {
      width: ${logoSize}px;
      height: ${logoSize}px;
      margin-bottom: ${Math.round((variant === "readme" ? 16 : 28) * scale)}px;
      filter: drop-shadow(0 0 ${Math.round(28 * scale)}px rgba(94, 234, 212, 0.22));
    }

    .logo-wrap svg {
      width: 100%;
      height: 100%;
      display: block;
    }

    .eyebrow {
      font-size: ${eyebrowSize}px;
      font-weight: 500;
      letter-spacing: 0.18em;
      text-transform: uppercase;
      color: #3a9e6e;
      margin-top: ${landingOrder ? Math.round(6 * scale) + "px" : "0"};
      margin-bottom: ${landingOrder ? "0" : Math.round(10 * scale) + "px"};
    }

    .title {
      font-size: ${titleSize}px;
      font-weight: 600;
      letter-spacing: -0.035em;
      line-height: 1;
      color: #f4f7fb;
      margin-bottom: ${landingOrder ? "0" : Math.round(18 * scale) + "px"};
    }

    .rule {
      width: ${Math.round(48 * scale)}px;
      height: 1px;
      background: linear-gradient(
        90deg,
        transparent,
        rgba(94, 234, 212, 0.55),
        transparent
      );
      margin-bottom: ${Math.round(18 * scale)}px;
    }

    .subtitle {
      font-size: ${subtitleSize}px;
      font-weight: 400;
      line-height: 1.35;
      letter-spacing: -0.01em;
      color: #94a3b8;
      max-width: ${Math.round(820 * scale)}px;
      margin-top: ${Math.round(14 * scale)}px;
    }

    .domain {
      position: absolute;
      bottom: ${Math.round(36 * scale)}px;
      right: ${Math.round(44 * scale)}px;
      font-size: ${domainSize}px;
      font-weight: 500;
      letter-spacing: 0.06em;
      color: rgba(148, 163, 184, 0.45);
    }
  </style>
</head>
<body>
  <div class="canvas">
    <div class="glow-primary" aria-hidden="true"></div>
    <div class="glow-accent" aria-hidden="true"></div>
    <div class="frame">
      <div class="logo-wrap" aria-hidden="true">
        <svg viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg">
          <path fill="#5eead4" d="${logoPath}" />
        </svg>
      </div>
      ${titleBlock}
      ${subtitle}
    </div>
    ${domainMarkup}
  </div>
</body>
</html>`
}

async function render() {
  const browser = await chromium.launch()
  const page = await browser.newPage()

  for (const { width, height, path, variant } of OUTPUTS) {
    mkdirSync(dirname(path), { recursive: true })
    await page.setViewportSize({ width, height })
    await page.setContent(buildHtml(width, height, variant), {
      waitUntil: "networkidle",
    })
    await page.waitForTimeout(120)
    const buffer = await page.screenshot({ type: "png", omitBackground: false })
    writeFileSync(path, buffer)
    console.log(`Wrote ${path} (${width}×${height}, ${variant})`)
  }

  await browser.close()
}

render().catch((err) => {
  console.error(err)
  process.exit(1)
})
