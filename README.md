# VibeGuard

Local security layer for AI coding agents (Cursor, Claude Code). VibeGuard intercepts **shell/terminal commands** and **MCP tool calls**, holding them for user approval via a terminal CLI and macOS alert-only notifications.

## Status

**MVP complete (vgf Phases 1–8):** shell/MCP intercept → YAML policy (auto-allow/deny) → macOS notify → terminal approvals → `vibeguard doctor` bypass detection → HTTP Stream MCP proxy library. **Tier 2 interactive UI** (`vibeguard ui`) is built into the binary. HTTP URL install wrap remains future work per [roadmap](docs/roadmap.md).

## Quick start

```bash
make build
vibeguard daemon start
vibeguard install          # wire Cursor/Claude hooks + MCP wrap
vibeguard status
```

When an agent is blocked, open the interactive approval UI:

```bash
vibeguard ui
```

- **↑/↓** or **j/k** — select a pending request
- **a** — approve · **d** — deny · **r** — refresh · **g** — toggle auto-approve · **q** — quit
- Auto-refreshes every ~2s while running

For hands-off local dev (policy/LLM checks still run; audit log kept):

```bash
vibeguard ui --auto-approve   # start with auto-approve on
vibeguard ui                  # press g to toggle auto-approve on/off
```

Pending items are allowed automatically on each refresh. This is session-only — it does not write `~/.vibeguard/policy.yaml` rules (use `vibeguard approve --always` for that).

`vibeguard ui` and other control-plane commands are **auto-allowed** by hooks so a Cursor agent can unblock itself. If that still fails, use **Terminal.app** (outside Cursor) or set `VIBEGUARD_DEV=1` for local dev/testing only (bypasses the hook queue and all policy checks entirely).

### Scripting / advanced

For automation or CI, use the raw CLI:

```bash
vibeguard pending --json
vibeguard approve          # auto-picks when one pending
vibeguard approve <id>
vibeguard deny             # auto-picks when one pending
vibeguard deny <id> --reason "too risky"
```

## Optional polish (gum / fzf)

You no longer need external tools for day-to-day approvals — use `vibeguard ui`. If you prefer gum/fzf pipelines:

```bash
brew install gum
vibeguard pending --json | jq -r '.[] | "\(.id)\t\(.command // .tool_name)"' | gum choose --header "Pending approvals"
```

macOS notifications are **alert-only** — decisions always happen in the terminal.

## Architecture

- **Single Go binary** — daemon, CLI, hook bridge, MCP wrap, and interactive TUI
- **Terminal-first UX** — `vibeguard ui` for keyboard-driven approvals; notifications are alert-only
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
| `http://127.0.0.1:9477/v1/health` | Daemon HTTP health |

## License

MIT — see [LICENSE](LICENSE).
