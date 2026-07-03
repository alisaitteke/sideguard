#!/usr/bin/env bash
# One-time cleanup after SideGuard rename (no legacy VibeGuard installs).
set -euo pipefail

UID_NUM="$(id -u)"
DOMAIN="gui/${UID_NUM}"
LAUNCH_AGENTS="${HOME}/Library/LaunchAgents"

info() { printf '→ %s\n' "$*"; }

for label in com.vibeguard.tray com.vibeguard.daemon; do
  plist="${LAUNCH_AGENTS}/${label}.plist"
  if launchctl print "${DOMAIN}/${label}" &>/dev/null; then
    info "Stopping ${label}…"
    launchctl bootout "${DOMAIN}/${label}" 2>/dev/null \
      || launchctl bootout "${DOMAIN}" "${plist}" 2>/dev/null \
      || true
  fi
  if [[ -f "${plist}" ]]; then
    info "Removing ${plist}"
    rm -f "${plist}"
  fi
done

if pgrep -f '/bin/vibeguard' &>/dev/null; then
  info "Stopping vibeguard processes…"
  pkill -f '/bin/vibeguard' || true
fi

if [[ -d "${HOME}/.vibeguard" ]]; then
  info "Removing ${HOME}/.vibeguard"
  rm -rf "${HOME}/.vibeguard"
fi

if [[ -f "$(cd "$(dirname "$0")/.." && pwd)/bin/vibeguard" ]]; then
  info "Removing repo bin/vibeguard"
  rm -f "$(cd "$(dirname "$0")/.." && pwd)/bin/vibeguard"
fi

info "Done. Run: sideguard install --cursor  (when ready to re-install hooks with sideguard)"
