/**
 * Landing page — composes all marketing sections inside semantic main.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
import { FeaturesSection } from "@/components/sections/FeaturesSection"
import { HeroSection } from "@/components/sections/HeroSection"
import { InstallSection } from "@/components/sections/InstallSection"
import { PlatformsSection } from "@/components/sections/PlatformsSection"
import { QuickStartSection } from "@/components/sections/QuickStartSection"

export function HomePage() {
  return (
    <main>
      <HeroSection />
      <FeaturesSection />
      <InstallSection />
      <QuickStartSection />
      <PlatformsSection />
    </main>
  )
}
