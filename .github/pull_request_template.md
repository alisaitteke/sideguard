## Summary

<!-- What changed and why (1–3 sentences) -->

## Type

- [ ] Bug fix
- [ ] Feature
- [ ] Documentation
- [ ] Website (`site/`)
- [ ] Chore / refactor

## Test plan

<!-- How you verified the change -->

- [ ] `make build` (use `CGO_ENABLED=1` if `internal/tray/` or tray CLI commands were touched)
- [ ] `go test ./...`
- [ ] Manual smoke (if daemon, hooks, MCP proxy, or install flow changed):
  - [ ] `sideguard daemon start` and `sideguard status`
  - [ ] N/A — no runtime behavior change

## Checklist

- [ ] No secrets, API keys, or `credentials.yaml` content in the diff
- [ ] User-facing CLI or install changes reflected in README (if applicable)
- [ ] Tray / macOS changes tested locally or marked N/A

## Related issues

<!-- Link issues: Fixes #123 -->
