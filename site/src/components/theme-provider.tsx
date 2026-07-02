/* eslint-disable react-refresh/only-export-components */
import * as React from "react"

type ResolvedTheme = "dark" | "light"

type ThemeProviderState = {
  theme: "system"
  resolvedTheme: ResolvedTheme
}

const COLOR_SCHEME_QUERY = "(prefers-color-scheme: dark)"

const ThemeProviderContext = React.createContext<
  ThemeProviderState | undefined
>(undefined)

function getSystemTheme(): ResolvedTheme {
  if (window.matchMedia(COLOR_SCHEME_QUERY).matches) {
    return "dark"
  }

  return "light"
}

function applySystemTheme() {
  const root = document.documentElement
  const resolvedTheme = getSystemTheme()

  root.classList.remove("light", "dark")
  root.classList.add(resolvedTheme)

  return resolvedTheme
}

/**
 * Applies `light` / `dark` on `<html>` from the OS color-scheme preference only.
 * No manual switcher, localStorage override, or keyboard toggle.
 */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [resolvedTheme, setResolvedTheme] = React.useState<ResolvedTheme>(() =>
    applySystemTheme()
  )

  React.useEffect(() => {
    const mediaQuery = window.matchMedia(COLOR_SCHEME_QUERY)
    const handleChange = () => {
      setResolvedTheme(applySystemTheme())
    }

    mediaQuery.addEventListener("change", handleChange)

    return () => {
      mediaQuery.removeEventListener("change", handleChange)
    }
  }, [])

  const value = React.useMemo(
    () => ({
      theme: "system" as const,
      resolvedTheme,
    }),
    [resolvedTheme]
  )

  return (
    <ThemeProviderContext.Provider value={value}>
      {children}
    </ThemeProviderContext.Provider>
  )
}

export const useTheme = () => {
  const context = React.useContext(ThemeProviderContext)

  if (context === undefined) {
    throw new Error("useTheme must be used within a ThemeProvider")
  }

  return context
}
