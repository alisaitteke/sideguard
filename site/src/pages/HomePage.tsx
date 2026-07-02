/**
 * Landing page: composes all marketing sections inside semantic main.
 */
import { FaqSection } from "@/components/sections/FaqSection"
import { FeaturesSection } from "@/components/sections/FeaturesSection"
import { HeroSection } from "@/components/sections/HeroSection"
import { InstallSection } from "@/components/sections/InstallSection"
import { NotWhatSection } from "@/components/sections/NotWhatSection"
import { PlatformsSection } from "@/components/sections/PlatformsSection"
import { PromptInjectionSection } from "@/components/sections/PromptInjectionSection"
import { QuickStartSection } from "@/components/sections/QuickStartSection"
import { SeoJsonLd } from "@/components/SeoJsonLd"

export function HomePage() {
  return (
    <main>
      <SeoJsonLd />
      <HeroSection />
      <PromptInjectionSection />
      <NotWhatSection />
      <FeaturesSection />
      <InstallSection />
      <QuickStartSection />
      <PlatformsSection />
      <FaqSection />
    </main>
  )
}
