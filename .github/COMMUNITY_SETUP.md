# GitHub community features — maintainer setup

## Done (remote / via `gh`)

| Item | Status |
| --- | --- |
| **Discussions** | Enabled |
| **Discussion categories** | `Q&A`, `Ideas`, `Show and tell`, `Announcements` (+ default General, Polls) |
| **Private vulnerability reporting** | Enabled (`gh api -X PUT repos/alisaitteke/sideguard/private-vulnerability-reporting`) |
| **Labels** | `triage`, `security`, `support`, `press` created (plus default GitHub labels) |

## After merging community template PR

Verify on GitHub:

- [New issue chooser](https://github.com/alisaitteke/sideguard/issues/new/choose) — no blank issue for contributors; templates + contact links
- [New discussion](https://github.com/alisaitteke/sideguard/discussions/new/choose) — Q&A, Ideas, Show and tell templates
- [Security → Report vulnerability](https://github.com/alisaitteke/sideguard/security/advisories/new)
- [SECURITY.md](https://github.com/alisaitteke/sideguard/blob/main/SECURITY.md) appears under Security → Policy (after `SECURITY.md` is on `main`)

## Manual only (no `gh` / API yet)

### Saved views (Issues page)

Create in the repository **Issues** sidebar → **Views** → **New view**:

| View name | Filter |
| --- | --- |
| Open bugs | `is:issue is:open label:bug` |
| Needs triage | `is:issue is:open label:triage` |

### Discussion moderation habits

When answering in **Q&A**:

- Use **Post as Admin** for official install/policy guidance
- **Verify answer** on the best reply when it is authoritative (e.g. reload steps after `install`)

### Optional later

- **GitHub Project** + auto-add workflow when backlog grows (issue form `projects:` needs reporter write access otherwise)
- **Dependabot / secret scanning** — separate Security settings; not required for community templates
