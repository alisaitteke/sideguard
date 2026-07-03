# Security Policy

## Supported versions

Security fixes are provided for:

| Version | Supported |
| --- | --- |
| Latest [GitHub Release](https://github.com/alisaitteke/sideguard/releases) | Yes |
| `main` branch (pre-release) | Best-effort for critical issues |

Older release tags may not receive patches. Upgrade to the latest release when possible.

## Reporting a vulnerability

**Do not** open a public GitHub issue with vulnerability details, proof-of-concept exploits, or live attack steps.

Use **GitHub Private Vulnerability Reporting**:

1. Go to the [Security tab](https://github.com/alisaitteke/sideguard/security) on this repository.
2. Click **Report a vulnerability**, or open [New security advisory](https://github.com/alisaitteke/sideguard/security/advisories/new) directly.
3. Submit a **private** draft advisory. Only maintainers and GitHub security staff can see it until coordinated disclosure.

An additional security contact email may be published here in the future. Until then, private reporting via GitHub is the primary channel.

## What to include

Help us triage quickly:

- **Affected version** — output of `sideguard version` or release tag
- **Component** — daemon, hook bridge, MCP proxy, install wiring, tray, update mechanism, etc.
- **Impact** — what an attacker could achieve (local privilege, policy bypass, data exposure, etc.)
- **Reproduction** — minimal steps if safe to share privately
- **Environment** — OS, AI client (Cursor / Claude Code), install method

Redact personal paths and secrets in reports.

## Response expectations

We aim to acknowledge new reports within **72 hours**. Resolution time depends on severity and complexity. We may ask for clarification or propose a coordinated disclosure timeline.

## Safe disclosure

Please give us reasonable time to investigate and release a fix before public disclosure. We appreciate responsible reports and will credit reporters when they wish to be named (unless you prefer anonymity).

## Scope notes

SideGuard runs locally on the user's machine (loopback daemon). Reports about **third-party AI clients** (Cursor, Claude Code) or **MCP servers you configure** may be out of scope unless SideGuard's integration introduces a distinct vulnerability.
