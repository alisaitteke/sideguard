# Contributing to SideGuard

Thank you for your interest in SideGuard. This guide covers local development, pull requests, and how to report issues.

## Prerequisites

- **Go** — version in [`go.mod`](go.mod) (currently Go 1.25+)
- **Make** — build and test shortcuts
- **CGO** (optional) — required for the menu bar tray (`CGO_ENABLED=1`); default in the Makefile

On macOS, tray development may need Xcode command-line tools. On Linux CI, `libayatana-appindicator3-dev` is used for systray builds (see [`.github/workflows/ci.yml`](.github/workflows/ci.yml)).

## Development setup

```bash
git clone https://github.com/alisaitteke/sideguard.git
cd sideguard
make build          # outputs bin/sideguard
make test           # go test ./...
make lint           # golangci-lint if installed
```

Run the daemon locally:

```bash
make run-daemon
# or: ./bin/sideguard daemon start
./bin/sideguard status
```

For tray work on macOS:

```bash
CGO_ENABLED=1 make build
./bin/sideguard tray
# or: CGO_ENABLED=1 make tray-app && open "dist/SideGuard Tray.app"
```

## Project map

SideGuard is a single Go binary (daemon, CLI, hooks, MCP proxy, terminal UI, optional tray) plus the marketing site under `site/`.

See [ARCHITECTURE.md](ARCHITECTURE.md) for component boundaries, request flows, and security model.

| Path | Responsibility |
| --- | --- |
| `cmd/sideguard/` | Cobra CLI commands |
| `internal/daemon/` | Loopback HTTP API, approval queue |
| `internal/hook/` | Cursor / Claude Code hook bridge |
| `internal/proxy/` | MCP STDIO proxy |
| `internal/policy/` | YAML policy evaluation |
| `internal/detect/` | Shell command auto-detect engine |
| `internal/store/` | SQLite audit and history |
| `internal/tray/` | macOS popover / systray |
| `site/` | [sideguard.io](https://sideguard.io) landing (Vite + React) |

## Pull requests

1. Branch from `main`.
2. Keep changes focused — one logical change per PR when possible.
3. Fill out the [pull request template](.github/pull_request_template.md).
4. Ensure CI passes: `make build` and `go test ./...`.
5. Update [README.md](README.md) if you change user-facing CLI behavior or install steps.

We review PRs as time allows. Smaller, well-tested diffs are easier to merge.

## Reporting issues

Use the [issue chooser](https://github.com/alisaitteke/sideguard/issues/new/choose) — blank issues are disabled.

| Need | Template |
| --- | --- |
| Reproducible bug | **Bug Report** — include `sideguard version`, `sideguard doctor`, and steps to reproduce |
| Install / setup help | **Installation Support** |
| Feature | **Feature Request** |
| Security vulnerability | [Private vulnerability reporting](https://github.com/alisaitteke/sideguard/security/advisories/new) — see [SECURITY.md](SECURITY.md) |
| Press / media | **Press Inquiry** |

For open questions or ideas, prefer [Discussions](https://github.com/alisaitteke/sideguard/discussions).

### Discussions from the terminal

With [GitHub CLI](https://cli.github.com/) v2.94+ and Discussions enabled:

```bash
gh discussion list --repo alisaitteke/sideguard
gh discussion view <number> --repo alisaitteke/sideguard
gh discussion create --repo alisaitteke/sideguard --title "..." --body "..."
gh discussion comment <number> --repo alisaitteke/sideguard --body "..."
```

## Security

Do **not** open public issues with exploit details. Follow [SECURITY.md](SECURITY.md).

## Code style

- Match existing patterns in the file you edit.
- Comments and docstrings in **English**.
- New Go files: MIT license header (see existing files in `cmd/sideguard/`).
- No secrets, API keys, or real `credentials.yaml` content in commits.
- Prefer simple, idiomatic Go over extra abstraction layers.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
