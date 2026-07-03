/**
 * Headless capture page for Playwright video export (Product Hunt gallery, social).
 * See site/scripts/render-prompt-injection-video.mjs
 */
import { ExportVideoWatermark } from "@/components/export/ExportVideoWatermark"
import { PromptInjectionScene } from "@/components/sections/PromptInjectionScene"
import "@/styles/prompt-injection-export.css"

const EXPORT_WIDTH = 1270
const EXPORT_HEIGHT = 760

type PromptInjectionExportPageProps = {
  clip?: "full" | "pan"
}

export function PromptInjectionExportPage({
  clip = "full",
}: PromptInjectionExportPageProps) {
  return (
    <div
      className="prompt-injection-export-root"
      style={{ width: EXPORT_WIDTH, height: EXPORT_HEIGHT }}
    >
      <PromptInjectionScene mode={clip === "pan" ? "export-pan" : "export"} />
      <ExportVideoWatermark />
    </div>
  )
}

export type { PromptInjectionExportState } from "@/components/sections/PromptInjectionScene"
