/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Download card for a single press asset with preview tile.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import { DownloadIcon } from "lucide-react"

import { buttonVariants } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import type { MediaKitAsset } from "@/lib/media-kit-assets"
import { cn } from "@/lib/utils"

const PREVIEW_BG: Record<NonNullable<MediaKitAsset["previewBg"]>, string> = {
  dark: "bg-[#10161c]",
  light: "bg-white",
  checkered:
    "bg-[linear-gradient(45deg,#e5e7eb_25%,transparent_25%),linear-gradient(-45deg,#e5e7eb_25%,transparent_25%),linear-gradient(45deg,transparent_75%,#e5e7eb_75%),linear-gradient(-45deg,transparent_75%,#e5e7eb_75%)] bg-size-[12px_12px] bg-position-[0_0,0_6px,6px_-6px,-6px_0px]",
}

type AssetCardProps = {
  asset: MediaKitAsset
}

export function AssetCard({ asset }: AssetCardProps) {
  const isSvg = asset.format === "SVG"
  const previewBg = asset.previewBg ?? "dark"

  return (
    <Card className="overflow-hidden">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">{asset.label}</CardTitle>
        <CardDescription>
          {asset.format}
          {asset.dimensions ? ` · ${asset.dimensions}` : ""}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div
          className={cn(
            "flex aspect-video items-center justify-center overflow-hidden rounded-lg border border-border p-6",
            PREVIEW_BG[previewBg]
          )}
        >
          <img
            src={asset.path}
            alt=""
            className={cn(
              "max-h-full max-w-full object-contain",
              isSvg ? "h-16 w-auto" : "h-full w-full"
            )}
            loading="lazy"
          />
        </div>
      </CardContent>
      <CardFooter>
        <a
          href={asset.path}
          download
          className={buttonVariants({ variant: "outline", size: "sm", className: "w-full" })}
        >
          <DownloadIcon />
          Download
        </a>
      </CardFooter>
    </Card>
  )
}
