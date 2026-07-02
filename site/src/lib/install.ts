/** Public install command strings for the landing page.
 *  Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md */
export const INSTALL_COMMAND =
  "curl -fsSL https://sideguard.io/setup.sh | sh"

export const FALLBACK_INSTALL_COMMAND =
  "curl -fsSL https://raw.githubusercontent.com/alisaitteke/sideguard/main/setup.sh | sh"

export const QUICK_START_COMMANDS = `sideguard daemon start
sideguard install          # Cursor/Claude hooks + MCP wrap + daemon (+ macOS tray)
sideguard status
sideguard clients reload   # reload hooks/MCP in Cursor & Claude Code`
