/**
 * Shared SideGuard brand tokens for static asset render scripts.
 * Keep in sync with --hero-logo in site/src/index.css.
 */
import { readFileSync } from "node:fs"
import { join, dirname } from "node:path"
import { fileURLToPath } from "node:url"

const __dirname = dirname(fileURLToPath(import.meta.url))
const siteRoot = join(__dirname, "..")

export const BRAND = {
  logoLight: "#0d9488",
  logoDark: "#5eead4",
  heroBackground: "#10161c",
  glowMint: "rgba(94, 234, 212, 0.22)",
  primary: "#3a9e6e",
  primaryDark: "#2d8659",
  foreground: "#e6edf3",
  foregroundMuted: "#94a3b8",
  fontFamily: '"Geist", system-ui, sans-serif',
}

/** Launch and directory copy — keep aligned with site/index.html meta. */
export const COPY = {
  name: "SideGuard",
  domain: "sideguard.io",
  url: "https://sideguard.io",
  eyebrow: "Vibe Coding Security Tool",
  taglinePh: "MCP guard with human approval for Cursor & Claude Code",
  taglineShort:
    "Approve shell commands and MCP tools before your AI runs them.",
  descriptionMeta:
    "SideGuard is a vibe coding security tool and MCP guard for Cursor and Claude Code. Human-in-the-loop approval for shell commands and MCP tools—not an MCP antivirus. YAML policy, fail-closed hooks, local audit trail.",
  descriptionPh:
    "SideGuard is a local MCP guard for Cursor and Claude Code. Before your AI assistant runs a shell command or MCP tool on your machine, SideGuard asks you to approve. YAML policy, fail-closed hooks, and a local audit trail—no cloud proxy, no MCP antivirus signatures.",
  installCommand: "curl -fsSL https://sideguard.io/setup.sh | sh",
  maker: {
    name: "Ali Sait Teke",
    url: "https://alisait.com",
    github: "https://github.com/alisaitteke",
    linkedin: "https://www.linkedin.com/in/alisait/",
    twitter: "https://x.com/alisaitteke",
  },
  links: {
    website: "https://sideguard.io",
    github: "https://github.com/alisaitteke/sideguard",
    install: "https://sideguard.io/setup.sh",
  },
}

/** Shield + checkmark path from public logo.svg (mask-friendly source of truth). */
export function readLogoPath() {
  const logoSvg = readFileSync(join(siteRoot, "public/assets/logo.svg"), "utf8")
  const match = logoSvg.match(/<path[^>]+d="([^"]+)"/)
  if (!match) {
    throw new Error("Could not extract logo path from public/assets/logo.svg")
  }
  return match[1]
}
