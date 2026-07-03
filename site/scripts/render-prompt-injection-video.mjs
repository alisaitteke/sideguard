/**
 * Records the prompt-injection demo animation to MP4 + GIF for Product Hunt / social.
 *
 * Prerequisites: `npm run build` output in site/dist, Playwright Chromium, ffmpeg on PATH.
 *
 * Outputs:
 * - media-kit/gallery/05-prompt-injection-1270x760.mp4 (full timeline)
 * - media-kit/gallery/05-prompt-injection-1270x760.gif (pan clip only)
 * - media-kit/gallery/05-prompt-injection-1270x760.webm (full, intermediate)
 *
 * Usage: npm run render:prompt-injection  (site/)
 */
import { mkdtempSync, mkdirSync, renameSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import { dirname, join, resolve } from "node:path"
import { fileURLToPath } from "node:url"
import { execSync, spawn, spawnSync } from "node:child_process"
import { chromium } from "playwright"

const __dirname = dirname(fileURLToPath(import.meta.url))
const siteRoot = resolve(__dirname, "..")
const repoRoot = resolve(siteRoot, "..")
const galleryDir = join(repoRoot, "media-kit", "gallery")

const WIDTH = 1270
const HEIGHT = 760
const PREVIEW_PORT = 4173
const OUTPUT_BASENAME = "05-prompt-injection-1270x760"
const OUTPUT_MP4 = join(galleryDir, `${OUTPUT_BASENAME}.mp4`)
const OUTPUT_GIF = join(galleryDir, `${OUTPUT_BASENAME}.gif`)
const OUTPUT_WEBM = join(galleryDir, `${OUTPUT_BASENAME}.webm`)

const GIF_FPS = 12

function hasFfmpeg() {
  try {
    execSync("ffmpeg -version", { stdio: "ignore" })
    return true
  } catch {
    return false
  }
}

function runBuild() {
  console.log("Building site…")
  execSync("npm run build", { cwd: siteRoot, stdio: "inherit" })
}

/**
 * @returns {import('node:child_process').ChildProcess}
 */
function startPreview(port) {
  const proc = spawn(
    "npx",
    ["vite", "preview", "--port", String(port), "--strictPort"],
    {
      cwd: siteRoot,
      stdio: ["ignore", "pipe", "pipe"],
    },
  )

  proc.stdout?.on("data", (chunk) => {
    process.stdout.write(chunk)
  })
  proc.stderr?.on("data", (chunk) => {
    process.stderr.write(chunk)
  })

  return proc
}

/**
 * @param {string} url
 * @param {number} timeoutMs
 */
async function waitForHttp(url, timeoutMs = 30_000) {
  const deadline = Date.now() + timeoutMs

  while (Date.now() < deadline) {
    try {
      const response = await fetch(url)
      if (response.ok) {
        return
      }
    } catch {
      // preview still starting
    }
    await new Promise((resolveWait) => setTimeout(resolveWait, 250))
  }

  throw new Error(`Preview server did not respond at ${url}`)
}

/**
 * @param {string} inputPath
 * @param {string} outputPath
 */
function convertToMp4(inputPath, outputPath) {
  const result = spawnSync(
    "ffmpeg",
    [
      "-y",
      "-i",
      inputPath,
      "-an",
      "-c:v",
      "libx264",
      "-pix_fmt",
      "yuv420p",
      "-movflags",
      "+faststart",
      outputPath,
    ],
    { stdio: "inherit" },
  )

  if (result.status !== 0) {
    throw new Error("ffmpeg failed")
  }
}

/**
 * @param {string} inputPath
 * @param {string} outputPath
 */
function convertToGif(inputPath, outputPath) {
  const filter = [
    `fps=${GIF_FPS}`,
    `scale=${WIDTH}:${HEIGHT}:flags=lanczos`,
    "split[s0][s1]",
    "[s0]palettegen=max_colors=256:stats_mode=diff[p]",
    "[s1][p]paletteuse=dither=bayer:bayer_scale=3",
  ].join(",")

  const result = spawnSync(
    "ffmpeg",
    ["-y", "-i", inputPath, "-an", "-vf", filter, outputPath],
    { stdio: "inherit" },
  )

  if (result.status !== 0) {
    throw new Error("ffmpeg GIF conversion failed")
  }
}

/**
 * @param {string} exportPath
 * @param {string} [outWebm]
 */
async function recordExportClip(exportPath, outWebm) {
  const videoDir = mkdtempSync(join(tmpdir(), "sideguard-pi-video-"))
  mkdirSync(galleryDir, { recursive: true })

  const browser = await chromium.launch()
  const context = await browser.newContext({
    viewport: { width: WIDTH, height: HEIGHT },
    recordVideo: {
      dir: videoDir,
      size: { width: WIDTH, height: HEIGHT },
    },
    deviceScaleFactor: 1,
    colorScheme: "dark",
  })

  const page = await context.newPage()
  await page.emulateMedia({ reducedMotion: "no-preference" })

  const targetUrl = exportPath
  console.log(`Recording ${targetUrl}…`)

  await page.goto(targetUrl, { waitUntil: "networkidle", timeout: 60_000 })

  await page.waitForFunction(
    () => window.__PROMPT_INJECTION_EXPORT__?.ready === true,
    { timeout: 60_000 },
  )

  const durationMs = await page.evaluate(
    () => window.__PROMPT_INJECTION_EXPORT__?.durationMs ?? 0,
  )
  console.log(`Animation ready (duration ~${Math.round(durationMs / 1000)}s)`)

  await page.waitForFunction(
    () => window.__PROMPT_INJECTION_EXPORT__?.done === true,
    { timeout: 120_000 },
  )

  await page.waitForTimeout(400)

  const video = page.video()
  await context.close()
  await browser.close()

  if (!video) {
    throw new Error("Playwright did not attach a video recorder")
  }

  const capturedWebm = await video.path()
  const destination = outWebm ?? join(videoDir, "clip.webm")
  renameSync(capturedWebm, destination)
  rmSync(videoDir, { recursive: true, force: true })

  return destination
}

/**
 * @param {string} baseUrl
 */
async function recordFullVideo(baseUrl) {
  const webm = await recordExportClip(
    `${baseUrl}/export/prompt-injection`,
    OUTPUT_WEBM,
  )
  console.log(`Wrote ${webm}`)

  if (!hasFfmpeg()) {
    console.warn(
      "ffmpeg not found — kept WebM only. Install ffmpeg and re-run for MP4/GIF.",
    )
    return
  }

  convertToMp4(OUTPUT_WEBM, OUTPUT_MP4)
  console.log(`Wrote ${OUTPUT_MP4}`)
}

/**
 * @param {string} baseUrl
 */
async function recordPanGif(baseUrl) {
  const panWebm = join(
    mkdtempSync(join(tmpdir(), "sideguard-pi-pan-")),
    "pan.webm",
  )

  await recordExportClip(`${baseUrl}/export/prompt-injection-pan`, panWebm)

  if (!hasFfmpeg()) {
    rmSync(dirname(panWebm), { recursive: true, force: true })
    return
  }

  convertToGif(panWebm, OUTPUT_GIF)
  console.log(`Wrote ${OUTPUT_GIF} (pan clip)`)
  rmSync(dirname(panWebm), { recursive: true, force: true })
}

async function main() {
  runBuild()

  const preview = startPreview(PREVIEW_PORT)
  const baseUrl = `http://127.0.0.1:${PREVIEW_PORT}`

  try {
    await waitForHttp(baseUrl)
    await recordFullVideo(baseUrl)
    await recordPanGif(baseUrl)
  } finally {
    preview.kill("SIGTERM")
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
