/**
 * Renders SideGuard media-kit PNGs for Product Hunt, social, and press use.
 *
 * Output root: media-kit/ (repo root)
 *
 * Usage: npm run render:media-kit  (site/)
 */
import { readFileSync, writeFileSync, mkdirSync, copyFileSync } from "node:fs"
import { dirname, join, resolve } from "node:path"
import { fileURLToPath } from "node:url"
import { chromium } from "playwright"

import { BRAND, COPY, readLogoPath } from "./brand-tokens.mjs"

const __dirname = dirname(fileURLToPath(import.meta.url))
const siteRoot = resolve(__dirname, "..")
const repoRoot = resolve(siteRoot, "..")
const kitRoot = join(repoRoot, "media-kit")

const geistWoff2 = readFileSync(
  join(
    siteRoot,
    "node_modules/@fontsource-variable/geist/files/geist-latin-wght-normal.woff2",
  ),
)
const geistBase64 = geistWoff2.toString("base64")
const logoPath = readLogoPath()

function fontFaceCss() {
  return `@font-face {
      font-family: "Geist";
      src: url(data:font/woff2;base64,${geistBase64}) format("woff2");
      font-weight: 100 900;
      font-style: normal;
    }`
}

function glowCss() {
  return `
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
    }`
}

function logoSvg(fill, sizePx) {
  return `<svg viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg" width="${sizePx}" height="${sizePx}">
    <path fill="${fill}" d="${logoPath}" />
  </svg>`
}

/**
 * @param {number} size
 * @param {string} fill
 * @param {string} background
 * @param {number} [iconScale]
 */
function buildSquareIconHtml(size, fill, background, iconScale = 0.72) {
  const iconSize = Math.round(size * iconScale)
  const transparent = background === "transparent"
  return `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8" /><style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    width: ${size}px; height: ${size}px; overflow: hidden;
    background: ${background};
    display: flex; align-items: center; justify-content: center;
  }
  .icon { width: ${iconSize}px; height: ${iconSize}px; display: block; }
</style></head><body>
  <div class="icon">${logoSvg(fill, iconSize)}</div>
</body></html>`
}

/**
 * @param {number} width
 * @param {number} height
 */
function buildHeroBannerHtml(width, height) {
  const scale = width / 1280
  const logoSize = Math.round(96 * scale)
  const titleSize = Math.round(58 * scale)
  const eyebrowSize = Math.round(13 * scale)
  const subtitleSize = Math.round(21 * scale)
  const domainSize = Math.round(14 * scale)

  return `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8" /><style>
  ${fontFaceCss()}
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    width: ${width}px; height: ${height}px; overflow: hidden;
    font-family: ${BRAND.fontFamily};
    background: ${BRAND.heroBackground}; color: ${BRAND.foreground};
    -webkit-font-smoothing: antialiased;
  }
  .canvas { position: relative; width: 100%; height: 100%;
    display: flex; align-items: center; justify-content: center; }
  ${glowCss()}
  .frame { position: relative; z-index: 1; display: flex; flex-direction: column;
    align-items: center; text-align: center; padding: 0 ${Math.round(64 * scale)}px; max-width: 92%; }
  .logo-wrap {
    width: ${logoSize}px; height: ${logoSize}px;
    margin-bottom: ${Math.round(28 * scale)}px;
    filter: drop-shadow(0 0 ${Math.round(28 * scale)}px ${BRAND.glowMint});
  }
  .logo-wrap svg { width: 100%; height: 100%; display: block; }
  .eyebrow { font-size: ${eyebrowSize}px; font-weight: 500; letter-spacing: 0.18em;
    text-transform: uppercase; color: ${BRAND.primary}; margin-bottom: ${Math.round(10 * scale)}px; }
  .title { font-size: ${titleSize}px; font-weight: 600; letter-spacing: -0.035em;
    line-height: 1; color: #f4f7fb; margin-bottom: ${Math.round(18 * scale)}px; }
  .rule { width: ${Math.round(48 * scale)}px; height: 1px;
    background: linear-gradient(90deg, transparent, rgba(94, 234, 212, 0.55), transparent);
    margin-bottom: ${Math.round(18 * scale)}px; }
  .subtitle { font-size: ${subtitleSize}px; font-weight: 400; line-height: 1.35;
    letter-spacing: -0.01em; color: ${BRAND.foregroundMuted}; max-width: ${Math.round(820 * scale)}px; }
  .domain { position: absolute; bottom: ${Math.round(36 * scale)}px; right: ${Math.round(44 * scale)}px;
    font-size: ${domainSize}px; font-weight: 500; letter-spacing: 0.06em;
    color: rgba(148, 163, 184, 0.45); }
</style></head><body>
  <div class="canvas">
    <div class="glow-primary" aria-hidden="true"></div>
    <div class="glow-accent" aria-hidden="true"></div>
    <div class="frame">
      <div class="logo-wrap" aria-hidden="true">${logoSvg(BRAND.logoDark, logoSize)}</div>
      <p class="eyebrow">${COPY.eyebrow}</p>
      <h1 class="title">${COPY.name}</h1>
      <div class="rule" aria-hidden="true"></div>
      <p class="subtitle">${COPY.taglinePh}</p>
    </div>
    <span class="domain">${COPY.domain}</span>
  </div>
</body></html>`
}

/**
 * @param {number} width
 * @param {number} height
 * @param {"approval" | "integrations" | "install"} variant
 */
function buildGallerySlideHtml(width, height, variant) {
  const scale = width / 1270
  const pad = Math.round(80 * scale)
  const headlineSize = Math.round(52 * scale)
  const bodySize = Math.round(24 * scale)
  const logoSize = Math.round(72 * scale)

  const slides = {
    approval: {
      headline: "Nothing runs until you say yes",
      body: "Human-in-the-loop approval for shell commands and MCP tool calls. Fail-closed hooks hold risky actions until you approve in the terminal or menu-bar tray.",
      bullets: [
        "YAML policy you control",
        "Local audit trail",
        "No cloud proxy",
      ],
    },
    integrations: {
      headline: "Built for vibe coding workflows",
      body: "SideGuard hooks into Cursor and Claude Code so your AI assistant can move fast—without running destructive commands unchecked.",
      bullets: ["Cursor hooks", "Claude Code MCP wraps", "macOS & Linux"],
    },
    install: {
      headline: "Install in one command",
      body: "Open source. Runs locally on your machine. Your policy and audit data stay on disk.",
      bullets: null,
      command: COPY.installCommand,
    },
  }

  const slide = slides[variant]
  const bulletsHtml = slide.bullets
    ? `<ul class="bullets">${slide.bullets.map((b) => `<li>${b}</li>`).join("")}</ul>`
    : ""
  const commandHtml = slide.command
    ? `<div class="terminal"><code>${slide.command}</code></div>`
    : ""

  return `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8" /><style>
  ${fontFaceCss()}
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    width: ${width}px; height: ${height}px; overflow: hidden;
    font-family: ${BRAND.fontFamily};
    background: ${BRAND.heroBackground}; color: ${BRAND.foreground};
    -webkit-font-smoothing: antialiased;
  }
  .canvas { position: relative; width: 100%; height: 100%; display: flex;
    align-items: center; padding: ${pad}px; gap: ${Math.round(48 * scale)}px; }
  ${glowCss()}
  .glow-primary { top: -30%; width: 90%; }
  .content { position: relative; z-index: 1; flex: 1; max-width: 72%; }
  .logo { width: ${logoSize}px; height: ${logoSize}px; margin-bottom: ${Math.round(28 * scale)}px;
    filter: drop-shadow(0 0 ${Math.round(20 * scale)}px ${BRAND.glowMint}); }
  .logo svg { width: 100%; height: 100%; display: block; }
  .headline { font-size: ${headlineSize}px; font-weight: 600; letter-spacing: -0.03em;
    line-height: 1.1; color: #f4f7fb; margin-bottom: ${Math.round(20 * scale)}px; }
  .body { font-size: ${bodySize}px; line-height: 1.45; color: ${BRAND.foregroundMuted};
    margin-bottom: ${Math.round(24 * scale)}px; max-width: ${Math.round(680 * scale)}px; }
  .bullets { list-style: none; display: flex; flex-direction: column; gap: ${Math.round(12 * scale)}px; }
  .bullets li { font-size: ${Math.round(20 * scale)}px; color: ${BRAND.logoDark};
    padding-left: ${Math.round(28 * scale)}px; position: relative; }
  .bullets li::before { content: "✓"; position: absolute; left: 0; color: ${BRAND.primary}; font-weight: 600; }
  .terminal { display: inline-block; background: #0b0f14; border: 1px solid #21262d;
    border-radius: ${Math.round(10 * scale)}px; padding: ${Math.round(16 * scale)}px ${Math.round(20 * scale)}px;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: ${Math.round(18 * scale)}px; color: #e6edf3; }
  .badge { position: absolute; bottom: ${pad}px; right: ${pad}px; z-index: 1;
    font-size: ${Math.round(14 * scale)}px; font-weight: 500; letter-spacing: 0.12em;
    text-transform: uppercase; color: rgba(148, 163, 184, 0.5); }
</style></head><body>
  <div class="canvas">
    <div class="glow-primary" aria-hidden="true"></div>
    <div class="glow-accent" aria-hidden="true"></div>
    <div class="content">
      <div class="logo" aria-hidden="true">${logoSvg(BRAND.logoDark, logoSize)}</div>
      <h2 class="headline">${slide.headline}</h2>
      <p class="body">${slide.body}</p>
      ${bulletsHtml}
      ${commandHtml}
    </div>
    <span class="badge">${COPY.domain}</span>
  </div>
</body></html>`
}

function writeLogoSvgs() {
  const logosDir = join(kitRoot, "logos")
  mkdirSync(logosDir, { recursive: true })

  const maskSvg = readFileSync(join(siteRoot, "public/assets/logo.svg"), "utf8")
  writeFileSync(join(logosDir, "logo-mask.svg"), maskSvg)

  for (const [name, fill] of [
    ["logo-teal-light.svg", BRAND.logoLight],
    ["logo-teal-dark.svg", BRAND.logoDark],
    ["logo-white.svg", "#ffffff"],
    ["logo-black.svg", "#0b0f14"],
  ]) {
    writeFileSync(
      join(logosDir, name),
      `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
  <path fill="${fill}" d="${logoPath}" />
</svg>
`,
    )
  }
}

function writeBrandJson() {
  writeFileSync(
    join(kitRoot, "brand-colors.json"),
    `${JSON.stringify({ brand: BRAND, copy: COPY }, null, 2)}\n`,
  )
}

const PNG_JOBS = [
  { dir: "logos", file: "thumbnail-240.png", w: 240, h: 240, html: () => buildSquareIconHtml(240, BRAND.logoDark, BRAND.heroBackground) },
  { dir: "logos", file: "logo-on-dark-512.png", w: 512, h: 512, html: () => buildSquareIconHtml(512, BRAND.logoDark, BRAND.heroBackground) },
  { dir: "logos", file: "logo-on-dark-1024.png", w: 1024, h: 1024, html: () => buildSquareIconHtml(1024, BRAND.logoDark, BRAND.heroBackground) },
  { dir: "logos", file: "logo-teal-light-512.png", w: 512, h: 512, html: () => buildSquareIconHtml(512, BRAND.logoLight, "transparent") },
  { dir: "logos", file: "logo-teal-dark-512.png", w: 512, h: 512, html: () => buildSquareIconHtml(512, BRAND.logoDark, "transparent") },
  { dir: "icons", file: "app-icon-180.png", w: 180, h: 180, html: () => buildSquareIconHtml(180, BRAND.logoDark, BRAND.heroBackground) },
  { dir: "icons", file: "app-icon-512.png", w: 512, h: 512, html: () => buildSquareIconHtml(512, BRAND.logoDark, BRAND.heroBackground) },
  { dir: "banners", file: "og-1200x630.png", w: 1200, h: 630, html: () => buildHeroBannerHtml(1200, 630) },
  { dir: "banners", file: "social-1280x640.png", w: 1280, h: 640, html: () => buildHeroBannerHtml(1280, 640) },
  { dir: "banners", file: "twitter-1600x900.png", w: 1600, h: 900, html: () => buildHeroBannerHtml(1600, 900) },
  { dir: "banners", file: "linkedin-1200x627.png", w: 1200, h: 627, html: () => buildHeroBannerHtml(1200, 627) },
  { dir: "gallery", file: "01-hero-1270x760.png", w: 1270, h: 760, html: () => buildHeroBannerHtml(1270, 760) },
  { dir: "gallery", file: "02-approval-1270x760.png", w: 1270, h: 760, html: () => buildGallerySlideHtml(1270, 760, "approval") },
  { dir: "gallery", file: "03-integrations-1270x760.png", w: 1270, h: 760, html: () => buildGallerySlideHtml(1270, 760, "integrations") },
  { dir: "gallery", file: "04-install-1270x760.png", w: 1270, h: 760, html: () => buildGallerySlideHtml(1270, 760, "install") },
]

async function render() {
  writeLogoSvgs()
  writeBrandJson()

  for (const sub of ["logos", "icons", "banners", "gallery"]) {
    mkdirSync(join(kitRoot, sub), { recursive: true })
  }

  const browser = await chromium.launch()
  const page = await browser.newPage()

  for (const job of PNG_JOBS) {
    const outPath = join(kitRoot, job.dir, job.file)
    const transparent = job.file.includes("teal-")
    await page.setViewportSize({ width: job.w, height: job.h })
    await page.setContent(job.html(), { waitUntil: "networkidle" })
    await page.waitForTimeout(100)
    const buffer = await page.screenshot({
      type: "png",
      omitBackground: transparent,
    })
    writeFileSync(outPath, buffer)
    console.log(`Wrote ${outPath} (${job.w}×${job.h})`)
  }

  await browser.close()

  const ogSrc = join(siteRoot, "public/assets/og-card.png")
  try {
    copyFileSync(ogSrc, join(kitRoot, "banners", "og-card-site.png"))
    console.log(`Copied ${ogSrc} → media-kit/banners/og-card-site.png`)
  } catch {
    console.warn("og-card.png not found — run render:social-card first for og-card-site.png")
  }
}

render().catch((err) => {
  console.error(err)
  process.exit(1)
})
