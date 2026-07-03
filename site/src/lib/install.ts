/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/** Public install command strings for the landing page.
 *  Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md */
export const INSTALL_COMMAND =
  "curl -fsSL https://sideguard.io/setup.sh | sh"

export const FALLBACK_INSTALL_COMMAND =
  "curl -fsSL https://raw.githubusercontent.com/alisaitteke/sideguard/main/setup.sh | sh"

/** GitHub source view for setup.sh — linked from /contact for transparent review before curl | sh. */
export const SETUP_SCRIPT_GITHUB_URL =
  "https://github.com/alisaitteke/sideguard/blob/main/setup.sh"

export const QUICK_START_COMMANDS = `sideguard daemon start
sideguard install          # Cursor/Claude hooks + MCP wrap + daemon (+ macOS tray)
sideguard status
sideguard clients reload   # reload hooks/MCP in Cursor & Claude Code`
