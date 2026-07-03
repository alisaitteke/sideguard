/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Static contact page: press, product support, and quick links.
 * Plan: docs/plans/2026-07-03-1105-media-contact-pages/
 */
import {
  ExternalLinkIcon,
  MailIcon,
  MessageCircleIcon,
  NewspaperIcon,
} from "lucide-react"
import { Link } from "react-router-dom"

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  AUTHOR_GITHUB_URL,
  AUTHOR_LINKEDIN_URL,
  AUTHOR_NAME,
  AUTHOR_SITE_URL,
  AUTHOR_TITLE,
} from "@/lib/author"
import { INSTALL_COMMAND, SETUP_SCRIPT_GITHUB_URL } from "@/lib/install"
import { usePageMeta } from "@/lib/page-meta"
import { GITHUB_URL, SITE_URL } from "@/lib/seo"

type ContactLinkProps = {
  href: string
  label: string
  description?: string
  external?: boolean
}

function ContactLink({ href, label, description, external = true }: ContactLinkProps) {
  const className =
    "group flex items-start justify-between gap-3 rounded-lg border border-transparent px-3 py-2.5 transition-colors hover:border-border hover:bg-muted/40"

  const content = (
    <>
      <div className="text-left">
        <p className="text-sm font-medium text-foreground group-hover:text-primary">
          {label}
        </p>
        {description ? (
          <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {external ? (
        <ExternalLinkIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
      ) : null}
    </>
  )

  if (external) {
    return (
      <a href={href} target="_blank" rel="noopener noreferrer" className={className}>
        {content}
      </a>
    )
  }

  return (
    <Link to={href} className={className}>
      {content}
    </Link>
  )
}

export function ContactPage() {
  usePageMeta(
    "Contact SideGuard — Press, Support & Partnerships",
    "Reach the SideGuard team for press inquiries, partnerships, and product support via GitHub, LinkedIn, or the media kit.",
    "/contact"
  )

  return (
    <main>
      <section className="border-b border-border bg-hero px-4 py-16 text-center sm:py-20">
        <div className="mx-auto max-w-2xl">
          <p className="text-sm font-medium tracking-[0.18em] text-primary uppercase">
            Get in touch
          </p>
          <h1 className="mt-3 text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
            Contact
          </h1>
          <p className="mx-auto mt-4 text-base text-muted-foreground sm:text-lg">
            Press, partnerships, and product questions — we route each channel to
            the right place.
          </p>
        </div>
      </section>

      <div className="mx-auto grid max-w-4xl gap-6 px-4 py-16 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <NewspaperIcon className="size-5 text-primary" />
              <CardTitle>Press &amp; partnerships</CardTitle>
            </div>
            <CardDescription>
              Interviews, launch coverage, and brand collaborations.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-1">
            <p className="mb-3 text-sm text-muted-foreground">
              <span className="font-medium text-foreground">{AUTHOR_NAME}</span>
              <br />
              {AUTHOR_TITLE}
            </p>
            <ContactLink
              href={AUTHOR_SITE_URL}
              label="alisait.com"
              description="Personal site and portfolio"
            />
            <ContactLink
              href={AUTHOR_LINKEDIN_URL}
              label="LinkedIn"
              description="Direct message for press inquiries"
            />
            <ContactLink
              href={AUTHOR_GITHUB_URL}
              label="GitHub"
              description={`@${AUTHOR_NAME.split(" ").pop()?.toLowerCase() ?? "alisaitteke"}`}
            />
            <ContactLink
              href="/media"
              label="Media kit"
              description="Logos, banners, and launch copy"
              external={false}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <MessageCircleIcon className="size-5 text-primary" />
              <CardTitle>Product support</CardTitle>
            </div>
            <CardDescription>
              Bug reports, feature requests, and installation help.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-1">
            <ContactLink
              href={`${GITHUB_URL}/issues`}
              label="GitHub Issues"
              description="Preferred channel for bugs and feature requests"
            />
            <ContactLink
              href={`${GITHUB_URL}/discussions`}
              label="GitHub Discussions"
              description="Questions, ideas, and community Q&A"
            />
            <ContactLink
              href={GITHUB_URL}
              label="Repository"
              description="Source code, docs, and releases"
            />
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <div className="flex items-center gap-2">
              <MailIcon className="size-5 text-primary" />
              <CardTitle>Quick links</CardTitle>
            </div>
            <CardDescription>Install, site, and brand resources.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-1 sm:grid-cols-2">
            <ContactLink href={SITE_URL} label="sideguard.io" description="Product website" />
            <ContactLink
              href={SETUP_SCRIPT_GITHUB_URL}
              label="Install script"
              description={`Review on GitHub before running: ${INSTALL_COMMAND}`}
            />
            <ContactLink
              href="/media"
              label="Media kit"
              description="Press assets and brand guidelines"
              external={false}
            />
            <ContactLink
              href={`${GITHUB_URL}/releases`}
              label="Releases"
              description="Pre-built binaries and checksums"
            />
          </CardContent>
        </Card>

        <div className="flex items-center justify-center gap-6 lg:col-span-2">
          <a
            href={AUTHOR_LINKEDIN_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-primary hover:underline"
          >
            LinkedIn
          </a>
          <a
            href={GITHUB_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-primary hover:underline"
          >
            GitHub
          </a>
        </div>
      </div>
    </main>
  )
}
