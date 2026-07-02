# VibeGuard

Local security layer for AI coding agents (Cursor, Claude Code). VibeGuard intercepts **shell/terminal commands** and **MCP tool calls**, holding them for user approval via the terminal CLI (`vibeguard ui`), an optional macOS menu-bar tray, and alert-only notifications.

## Status

**MVP complete (vgf Phases 1–8):** shell/MCP intercept → YAML policy (auto-allow/deny) → macOS notify → terminal approvals → `vibeguard doctor` bypass detection → HTTP Stream MCP proxy library. **Tier 2 interactive UI** (`vibeguard ui`) is built into the binary. **Recent:** global approval mode (`vibeguard mode`), experimental macOS menu-bar tray (`vibeguard tray`), surgical `vibeguard uninstall`, client reload hints (`vibeguard clients reload`), and repo-scoped dev workspace policy (`vibeguard policy init-dev` / `install --dev`). **New:** local obfuscation-resistant shell auto-detect engine with smart-triage `auto` mode (`vibeguard mode set auto`) and persisted command history (`vibeguard history`). **GitHub Releases self-update** — tray background checks plus `vibeguard update` CLI (see [Updating](#updating)). HTTP URL install wrap remains future work per [roadmap](docs/roadmap.md).

## Quick start

```bash
make build
vibeguard daemon start
vibeguard install          # wire Cursor/Claude hooks + MCP wrap + daemon + macOS tray
vibeguard status
vibeguard clients reload   # how to reload hooks/MCP in Cursor & Claude Code
vibeguard tray             # macOS menu bar — Allow/Deny from the icon (experimental)
```

After `install` or `uninstall`, reload AI clients so hook and MCP config changes take effect. VibeGuard cannot force a reload:

| Client | Usually enough | If changes do not apply |
| --- | --- | --- |
| **Cursor** | Save `hooks.json` (auto-reload on save) | **Cmd+Shift+P** → **Developer: Reload Window** (no full quit) |
| **Claude Code** | Wait a few seconds (file watcher on `settings.json`) | `/exit` and start a new session; `/hooks` lists hooks but does not reload |

Run `vibeguard clients reload` for the full per-client guide.

On macOS, `vibeguard install` also registers the menu-bar tray LaunchAgent (`com.vibeguard.tray.plist`) so approvals are available from the menu bar after login. Use `--headless` to skip tray install (SSH, CI, or servers without a GUI session). `--skip-daemon` only skips the daemon LaunchAgent; hooks/MCP are unchanged.

To remove integration: `vibeguard uninstall` surgically strips VibeGuard hooks and MCP wraps (your other config stays). It removes daemon and tray LaunchAgents on macOS unless you pass `--keep-daemon`. Use `--restore-backup` to revert files from the oldest pre-install backup instead. Then run `vibeguard clients reload`.

### Menu bar tray (macOS, experimental)

Background menu-bar icon that polls the daemon on loopback (`127.0.0.1:9477`) every ~2s. The tray does **not** replace the terminal UI (`vibeguard ui`) or CLI (`approve` / `deny`).

**Prerequisites**

1. Daemon running: `vibeguard daemon start` (or `vibeguard daemon install-service` for login auto-start)
2. Build with CGO: `CGO_ENABLED=1 make build`

**Run**

| Method | Command |
| --- | --- |
| Terminal | `./bin/vibeguard tray` |
| `.app` bundle | `CGO_ENABLED=1 make tray-app` then `open "dist/VibeGuard Tray.app"` |

The `.app` bundle sets `LSUIElement` so the tray appears in the menu bar only (no Dock icon). Unsigned builds may be blocked by Gatekeeper — right-click the app → **Open** the first time.

**Login auto-start**

- **Default:** `vibeguard install` registers both the daemon and tray LaunchAgents on macOS.
- **Tray only:** `vibeguard tray install-service` — writes `~/Library/LaunchAgents/com.vibeguard.tray.plist` without re-running full install.
- **`.app` bundle:** System Settings → General → Login Items — add `VibeGuard Tray.app`, or use `CGO_ENABLED=1 make tray-app`.

**Popover panel (macOS)**

Click the menu-bar icon to open a **popover panel** below the icon (not a context menu). Each pending row has flat **Allow** and **Deny** buttons — no submenus. The panel shows daemon health, pending count, **Mode** (Ask / Auto / Auto-allow / Auto-deny segmented control), and up to **10** pending rows; use `vibeguard ui` for more. When new pending approvals arrive, the panel **auto-opens** if hidden. Click the icon again to dismiss.

On non-macOS builds, the tray uses a classic systray context menu instead.

When pending approvals exist, the icon switches to an orange-badge variant and the menu-bar title shows the count (updates on the ~2s poll, not push/instant).

If the daemon is not running at tray launch, status shows unreachable until you start the daemon.

When an agent is blocked, open the interactive approval UI:

```bash
vibeguard ui
```

- **↑/↓** or **j/k** — select a pending request
- **a** — approve · **d** — deny · **r** — refresh · **g** — cycle approval mode · **q** — quit
- Auto-refreshes every ~2s while running

Global approval mode (`ask` / `auto` / `auto-allow` / `auto-deny`) is persisted by the daemon and shared with the menu-bar tray:

```bash
vibeguard mode                    # show current mode
vibeguard mode set auto           # smart triage: safe pass, risky blocked, uncertain queue (default on new installs)
vibeguard mode set auto-allow     # hands-off local dev (audit logged)
vibeguard mode set auto-deny      # reject queued items (audit logged)
vibeguard mode set ask            # back to manual approvals
```

Every intercept decision is persisted locally — query it with `vibeguard history [--since 7d] [--denied] [--json] [search TERM]`.

Press **g** in `vibeguard ui` to cycle modes. Auto modes decide queued requests server-side (existing pending included). YAML policy deny rules still block at the hook before items reach the queue.

`vibeguard ui` and other control-plane commands are **auto-allowed** by hooks so a Cursor agent can unblock itself. If that still fails, use **Terminal.app** (outside Cursor) or set `VIBEGUARD_DEV=1` for local dev/testing only (bypasses the hook queue and all policy checks entirely).

### Developing VibeGuard inside Cursor

After `vibeguard install`, agent shell commands (`make`, `go test`, scripts) queue for approval — the agent cannot test the project without deadlocking. Use one of:

```bash
# Repo-scoped (recommended): allow make/go/scripts only under this repo
vibeguard policy init-dev
# or: vibeguard install --dev

# Full bypass for all commands in the agent environment (local only)
# Cursor: Settings → Agents → Environment → VIBEGUARD_DEV=1
# Terminal.app export does NOT apply to in-IDE agents.
```

Workspace dev policy is written to `.vibeguard/policy.yaml` (gitignored) and does not weaken global policy for other projects.

### Scripting / advanced

For automation or CI, use the raw CLI:

```bash
vibeguard pending --json
vibeguard approve          # auto-picks when one pending
vibeguard approve <id>
vibeguard deny             # auto-picks when one pending
vibeguard deny <id> --reason "too risky"
```

## Installing from GitHub Releases

Pre-built binaries are published on [GitHub Releases](https://github.com/alisaitteke/vibeguard/releases) with a `checksums.txt` (SHA256) for each tag. Pick the archive for your platform:

| Platform | Archive name pattern |
| --- | --- |
| macOS Apple Silicon | `vibeguard_<version>_darwin_arm64.tar.gz` |
| macOS Intel | `vibeguard_<version>_darwin_amd64.tar.gz` |
| Linux amd64 | `vibeguard_<version>_linux_amd64.tar.gz` |
| Linux arm64 | `vibeguard_<version>_linux_arm64.tar.gz` |
| Windows amd64 | `vibeguard_<version>_windows_amd64.zip` |

```bash
# Example (macOS arm64) — replace <version> with the release tag without "v"
VERSION=0.1.0
curl -fsSL -O "https://github.com/alisaitteke/vibeguard/releases/download/v${VERSION}/checksums.txt"
curl -fsSL -O "https://github.com/alisaitteke/vibeguard/releases/download/v${VERSION}/vibeguard_${VERSION}_darwin_arm64.tar.gz"
shasum -a 256 -c checksums.txt   # Linux: sha256sum -c checksums.txt
tar -xzf "vibeguard_${VERSION}_darwin_arm64.tar.gz"
sudo install -m 755 vibeguard /usr/local/bin/vibeguard   # or any directory on your PATH
vibeguard --version
vibeguard install
```

Release builds are **unsigned**. On macOS, Gatekeeper may quarantine the binary after download — see [macOS Gatekeeper](#macos-gatekeeper-unsigned-releases) below.

## Updating

VibeGuard checks [GitHub Releases](https://github.com/alisaitteke/vibeguard/releases) for newer versions, verifies SHA256 checksums before replacing the running binary, and keeps `~/.vibeguard` state (hooks, policy, audit DB) unchanged.

### Tray (background check)

When the menu-bar tray is running (`vibeguard install` on macOS, or `vibeguard tray` / systray on Linux/Windows), a **separate** background loop (default every **6 hours**) compares your binary version against the latest release. When an update is available:

- **macOS popover** — footer shows **Install update vX.Y.Z**; click to apply.
- **Linux / Windows systray** — **Install update vX.Y.Z…** menu item appears above Quit.

Install is **one-click and user-initiated** — nothing auto-applies without your action. The tray spawns `vibeguard update apply --restart --yes`, exits so the binary can be swapped, then the daemon and tray are restarted.

### CLI

```bash
vibeguard update check              # compare running version vs latest release
vibeguard update check --json       # machine-readable output
vibeguard update status             # last check time, latest known, background check state
vibeguard update apply              # download, verify checksum, replace current binary
vibeguard update apply --restart    # also restart daemon + tray after swap
vibeguard update apply --yes        # skip confirmation (scripts / tray)
vibeguard update apply --version 1.2.3   # install a specific release
```

Update metadata is stored in `~/.vibeguard/update-state.json`.

### Configuration

In `~/.vibeguard/config.yaml`:

```yaml
update:
  enabled: true          # false disables background tray checks and is reflected in update status
  check_interval: 6h     # tray poll interval (Go duration string)
  channel: stable        # reserved for future use; no effect in v1
```

### Dev builds

Local `make build` binaries embed `Version=dev` (or a `snapshot` tag). Background update checks are **skipped** for dev/snapshot builds so local development is not interrupted. Use `make build` / `go build` for hacking; use [GitHub Releases](#installing-from-github-releases) or `vibeguard update apply` for production installs.

### macOS Gatekeeper (unsigned releases)

Release binaries are not Apple-notarized. After download, macOS may block execution or show **“cannot be opened because the developer cannot be verified.”** Options:

1. **First launch** — right-click the binary (or `VibeGuard Tray.app`) → **Open** → confirm once.
2. **Remove quarantine** (if downloaded via browser):

```bash
xattr -dr com.apple.quarantine /path/to/vibeguard
```

Self-update (`vibeguard update apply`) downloads over HTTPS and verifies SHA256 against the release `checksums.txt` before replacing the binary.

## Optional polish (gum / fzf)

You no longer need external tools for day-to-day approvals — use `vibeguard ui`. If you prefer gum/fzf pipelines:

```bash
brew install gum
vibeguard pending --json | jq -r '.[] | "\(.id)\t\(.command // .tool_name)"' | gum choose --header "Pending approvals"
```

macOS notifications are **alert-only** — decisions always happen in the terminal.

## Architecture

- **Single Go binary** — daemon, CLI, hook bridge, MCP wrap, interactive TUI, and menu-bar tray (CGO on macOS)
- **Terminal-first UX** — `vibeguard ui` for keyboard-driven approvals; optional macOS menu-bar tray for Allow/Deny; notifications are alert-only
- **Hybrid interception** — MCP STDIO proxy + Cursor/Claude hook bridge
- **Fail-closed** — commands do not reach the OS until explicitly approved
- **LaunchAgent daemon** — user-session GUI context for `osascript` / `terminal-notifier`

## Documentation

| Document | Description |
| --- | --- |
| [docs/roadmap.md](docs/roadmap.md) | Product roadmap and API contracts |
| [docs/research-report.md](docs/research-report.md) | Cursor/Claude hooks and hybrid architecture research |
| [docs/integration-and-terminal-ui.md](docs/integration-and-terminal-ui.md) | Install flow and terminal approval UX |
| [docs/plans/2026-07-01-0127-vibeguard-foundation/](docs/plans/2026-07-01-0127-vibeguard-foundation/) | Phased implementation plan |

## Local paths

| Path | Purpose |
| --- | --- |
| `~/.vibeguard/run/vibeguard.sock` | Unix socket |
| `~/.vibeguard/audit.db` | SQLite audit log |
| `~/.vibeguard/update-state.json` | Last update check + latest known release |
| `http://127.0.0.1:9477/v1/health` | Daemon HTTP health |

## License

MIT — see [LICENSE](LICENSE).
