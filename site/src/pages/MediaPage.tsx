/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Enterprise press & brand assets page with downloadable media-kit files.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import { DownloadIcon } from "lucide-react"
import { Link } from "react-router-dom"

import { AssetCard } from "@/components/media/AssetCard"
import { CopyButton } from "@/components/media/CopyButton"
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import { buttonVariants } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import {
  BOILERPLATE,
  BRAND_COLORS,
  BRAND_GUIDELINES,
  GALLERY_VIDEO,
  LAUNCH_COPY_BLOCKS,
  MEDIA_KIT_ASSETS,
  MEDIA_KIT_ZIP_URL,
  META_BOILERPLATE,
  type MediaKitAsset,
} from "@/lib/media-kit-assets"
import { usePageMeta } from "@/lib/page-meta"
import { buildOrganizationJsonLd, GITHUB_URL, SITE_URL } from "@/lib/seo"

function JsonLdScript({ data }: { data: object }) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  )
}

function SectionHeading({
  id,
  title,
  description,
}: {
  id: string
  title: string
  description?: string
}) {
  return (
    <div id={id} className="scroll-mt-20">
      <h2 className="text-xl font-semibold tracking-tight sm:text-2xl">{title}</h2>
      {description ? (
        <p className="mt-2 max-w-2xl text-sm text-muted-foreground sm:text-base">
          {description}
        </p>
      ) : null}
    </div>
  )
}

function AssetGrid({ category }: { category: MediaKitAsset["category"] }) {
  const assets = MEDIA_KIT_ASSETS.filter((a) => a.category === category)
  return (
    <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {assets.map((asset) => (
        <AssetCard key={asset.path} asset={asset} />
      ))}
    </div>
  )
}

// Re-import type for AssetGrid — removed, imported above

export function MediaPage() {
  usePageMeta(
    "SideGuard Media Kit — Logos, Banners & Press Assets",
    "Download SideGuard logos, social banners, product screenshots, and launch copy for press, Product Hunt, and partner use.",
    "/media"
  )

  const mediaJsonLd = {
    ...buildOrganizationJsonLd(),
    mainEntityOfPage: `${SITE_URL}/media`,
  }

  return (
    <main>
      <JsonLdScript data={mediaJsonLd} />

      <section className="border-b border-border bg-hero px-4 py-16 text-center sm:py-20">
        <div className="mx-auto max-w-3xl">
          <p className="text-sm font-medium tracking-[0.18em] text-primary uppercase">
            Press &amp; Brand
          </p>
          <h1 className="mt-3 text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
            Media kit
          </h1>
          <p className="mx-auto mt-4 max-w-xl text-base text-muted-foreground sm:text-lg">
            Official logos, banners, product screenshots, and launch copy for
            journalists, partners, and launch platforms.
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            <a
              href={MEDIA_KIT_ZIP_URL}
              download
              className={buttonVariants({ size: "default" })}
            >
              <DownloadIcon />
              Download all assets
            </a>
            <Link to="/contact" className={buttonVariants({ variant: "outline" })}>
              Press inquiries
            </Link>
          </div>
        </div>
      </section>

      <div className="mx-auto max-w-5xl space-y-16 px-4 py-16">
        <section>
          <SectionHeading
            id="boilerplate"
            title="Boilerplate"
            description="Short description for directories, pitch decks, and partner listings."
          />
          <Card className="mt-6">
            <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
              <div>
                <CardTitle className="text-base">One-liner</CardTitle>
                <CardDescription>Product name + positioning</CardDescription>
              </div>
              <CopyButton text={BOILERPLATE} />
            </CardHeader>
            <CardContent>
              <p className="text-sm leading-relaxed text-foreground">{BOILERPLATE}</p>
            </CardContent>
          </Card>
          <Card className="mt-4">
            <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
              <div>
                <CardTitle className="text-base">Meta description</CardTitle>
                <CardDescription>SEO and directory listings</CardDescription>
              </div>
              <CopyButton text={META_BOILERPLATE} label="Copy meta" />
            </CardHeader>
            <CardContent>
              <p className="text-sm leading-relaxed text-muted-foreground">
                {META_BOILERPLATE}
              </p>
            </CardContent>
          </Card>
        </section>

        <section>
          <SectionHeading
            id="brand-colors"
            title="Brand colors"
            description="Teal accent on dark hero backgrounds. Geist Variable is the brand typeface."
          />
          <div className="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {BRAND_COLORS.map((color) => (
              <Card key={color.hex}>
                <CardContent className="flex items-center gap-4 pt-6">
                  <span
                    aria-hidden
                    className="size-12 shrink-0 rounded-lg border border-border shadow-sm"
                    style={{ backgroundColor: color.hex }}
                  />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium">{color.token}</p>
                    <p className="font-mono text-xs text-muted-foreground">{color.hex}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{color.usage}</p>
                  </div>
                  <CopyButton text={color.hex} label={`Copy ${color.hex}`} />
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        <section>
          <SectionHeading
            id="logos"
            title="Logos"
            description="Vector and raster marks for light and dark backgrounds."
          />
          <AssetGrid category="logo" />
        </section>

        <section>
          <SectionHeading
            id="icons"
            title="App icons"
            description="Square marks for directories and app-style placements."
          />
          <AssetGrid category="icon" />
        </section>

        <section>
          <SectionHeading
            id="banners"
            title="Social banners"
            description="Pre-sized headers for Open Graph, Twitter/X, and LinkedIn."
          />
          <AssetGrid category="banner" />
        </section>

        <section>
          <SectionHeading
            id="gallery"
            title="Product gallery"
            description="Product Hunt gallery slides and demo video (1270×760)."
          />
          <AssetGrid category="gallery" />
          <Card className="mt-6 overflow-hidden">
            <CardHeader>
              <CardTitle className="text-base">{GALLERY_VIDEO.label}</CardTitle>
              <CardDescription>MP4 · {GALLERY_VIDEO.dimensions}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <video
                className="w-full rounded-lg border border-border"
                controls
                playsInline
                preload="metadata"
                poster={GALLERY_VIDEO.poster}
              >
                <source src={GALLERY_VIDEO.mp4} type="video/mp4" />
              </video>
              <a
                href={GALLERY_VIDEO.mp4}
                download
                className="inline-flex items-center gap-2 text-sm text-primary hover:underline"
              >
                <DownloadIcon className="size-4" />
                Download MP4
              </a>
            </CardContent>
          </Card>
        </section>

        <section>
          <SectionHeading
            id="launch-copy"
            title="Launch copy"
            description="Ready-to-paste blocks for Product Hunt, social, and Show HN."
          />
          <Accordion className="mt-6 w-full">
            {LAUNCH_COPY_BLOCKS.map((block) => (
              <AccordionItem key={block.id} value={block.id}>
                <AccordionTrigger>{block.title}</AccordionTrigger>
                <AccordionContent>
                  <div className="flex flex-col gap-3">
                    <pre className="overflow-x-auto rounded-lg border border-border bg-muted/40 p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap text-foreground">
                      {block.text}
                    </pre>
                    <CopyButton text={block.text} className="self-start" />
                  </div>
                </AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        </section>

        <section>
          <SectionHeading
            id="guidelines"
            title="Logo usage"
            description="Keep the mark consistent across press and partner materials."
          />
          <ul className="mt-6 list-disc space-y-2 pl-5 text-sm text-muted-foreground">
            {BRAND_GUIDELINES.map((rule) => (
              <li key={rule}>{rule}</li>
            ))}
          </ul>
        </section>

        <Separator />

        <section className="rounded-xl border border-border bg-muted/30 px-6 py-8 text-center">
          <h2 className="text-lg font-semibold">Press inquiries</h2>
          <p className="mx-auto mt-2 max-w-md text-sm text-muted-foreground">
            For interviews, partnerships, or custom asset requests, reach out via
            our contact page.
          </p>
          <div className="mt-5 flex flex-wrap justify-center gap-3">
            <Link to="/contact" className={buttonVariants()}>
              Contact
            </Link>
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noopener noreferrer"
              className={buttonVariants({ variant: "outline" })}
            >
              GitHub
            </a>
          </div>
        </section>
      </div>
    </main>
  )
}
