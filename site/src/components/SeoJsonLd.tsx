/**
 * Injects JSON-LD structured data for search engines and AI crawlers that execute JS.
 */
import {
  buildFaqPageJsonLd,
  buildOrganizationJsonLd,
  buildSoftwareApplicationJsonLd,
  buildWebSiteJsonLd,
} from "@/lib/seo"

function JsonLdScript({ data }: { data: object }) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  )
}

export function SeoJsonLd() {
  return (
    <>
      <JsonLdScript data={buildWebSiteJsonLd()} />
      <JsonLdScript data={buildOrganizationJsonLd()} />
      <JsonLdScript data={buildSoftwareApplicationJsonLd()} />
      <JsonLdScript data={buildFaqPageJsonLd()} />
    </>
  )
}
