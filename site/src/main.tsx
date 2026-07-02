import { StrictMode } from "react"
import { createRoot } from "react-dom/client"

import "./index.css"
import { ThemeProvider } from "@/components/theme-provider.tsx"
import { Toaster } from "@/components/ui/sonner.tsx"
import { initPostHog } from "@/lib/posthog.ts"
import { AppRouter } from "@/router.tsx"

initPostHog()

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <AppRouter />
      <Toaster />
    </ThemeProvider>
  </StrictMode>
)
