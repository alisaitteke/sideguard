/**
 * Subtle corner branding for Playwright-captured promo videos (export route only).
 * See site/scripts/render-prompt-injection-video.mjs
 */
export function ExportVideoWatermark() {
  return (
    <div className="export-video-watermark" aria-hidden>
      <div className="export-video-watermark__logo" />
      <span className="export-video-watermark__name">SideGuard</span>
    </div>
  )
}
