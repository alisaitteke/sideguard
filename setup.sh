#!/usr/bin/env sh
# Copyright (c) 2026 Ali Sait Teke
# SPDX-License-Identifier: MIT
# SideGuard installer — download a release binary from GitHub or build from source.
# Usage:
#   curl -fsSL https://sideguard.io/setup.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/alisaitteke/sideguard/main/setup.sh | sh  # fallback
#   ./setup.sh
#
# Optional environment variables:
#   SIDEGUARD_INSTALL_MODE=github|source   (default: prompt when interactive, github when piped)
#   SIDEGUARD_VERSION=latest|v0.1.2|0.1.2   (default: latest; GitHub download only)
#   SIDEGUARD_INSTALL_DIR=/usr/local/bin      (default: /usr/local/bin)
#   SIDEGUARD_RUN_INSTALL=1                 (default: 0 — run `sideguard install` after binary install)

set -eu
(set -o pipefail) 2>/dev/null && set -o pipefail

GITHUB_OWNER="${GITHUB_OWNER:-alisaitteke}"
GITHUB_REPO="${GITHUB_REPO:-sideguard}"
GITHUB_CLONE_URL="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}.git"
DEFAULT_INSTALL_DIR="/usr/local/bin"
GO_MODULE="github.com/alisaitteke/sideguard"

info() {
  printf '%s\n' "$*" >&2
}

die() {
  info "error: $*"
  exit 1
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    die "required command not found: $1"
  fi
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    die "neither sha256sum nor shasum is available for checksum verification"
  fi
}

normalize_tag() {
  tag="$1"
  case "$tag" in
    v*) printf '%s' "$tag" ;;
    *) printf 'v%s' "$tag" ;;
  esac
}

resolve_latest_tag() {
  url_effective=$(curl -fsSL -o /dev/null -w '%{url_effective}' \
    "https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest") \
    || die "no GitHub releases found for ${GITHUB_OWNER}/${GITHUB_REPO} — publish a release or run: SIDEGUARD_INSTALL_MODE=source curl -fsSL https://sideguard.io/setup.sh | sh"
  tag="${url_effective##*/}"
  if [ -z "$tag" ] || [ "$tag" = "latest" ]; then
    die "could not parse latest release tag from GitHub redirect"
  fi
  printf '%s' "$tag"
}

detect_platform() {
  raw_os=$(uname -s 2>/dev/null || true)
  raw_arch=$(uname -m 2>/dev/null || true)

  case "$raw_os" in
    Darwin) os=darwin ;;
    Linux) os=linux ;;
    MINGW* | MSYS* | CYGWIN* | Windows*)
      os=windows
      ;;
    *)
      die "unsupported operating system: ${raw_os:-unknown} (supported: macOS, Linux)"
      ;;
  esac

  case "$raw_arch" in
    x86_64 | amd64) arch=amd64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *)
      die "unsupported CPU architecture: ${raw_arch:-unknown} (supported: amd64, arm64)"
      ;;
  esac

  if [ "$os" = "windows" ]; then
    info "Windows is not supported by this shell installer."
    info "Download the zip manually from:"
    info "  https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest"
    info "Look for: sideguard_<version>_windows_amd64.zip"
    exit 1
  fi
}

verify_checksum() {
  archive_name="$1"
  archive_path="$2"
  checksums_path="$3"

  if [ ! -f "$checksums_path" ]; then
    die "checksums file missing at $checksums_path"
  fi

  expected=""
  while IFS= read -r line || [ -n "$line" ]; do
    line=$(printf '%s' "$line" | tr -d '\r')
    [ -n "$line" ] || continue
    hash=$(printf '%s' "$line" | awk '{print $1}')
    name=$(printf '%s' "$line" | awk '{print $NF}' | tr -d '*')
    if [ "$name" = "$archive_name" ]; then
      expected="$hash"
      break
    fi
  done < "$checksums_path"

  if [ -z "$expected" ]; then
    die "no checksum entry found for ${archive_name} in checksums.txt"
  fi

  actual=$(sha256_file "$archive_path")
  if [ "$actual" != "$expected" ]; then
    die "checksum mismatch for ${archive_name}"
  fi
}

install_binary() {
  src="$1"
  dest_dir="$2"
  dest="${dest_dir}/sideguard"

  if [ ! -d "$dest_dir" ]; then
    if command -v sudo >/dev/null 2>&1 && [ "$(id -u)" -ne 0 ]; then
      info "Creating ${dest_dir} (may prompt for sudo)..."
      sudo mkdir -p "$dest_dir"
    else
      mkdir -p "$dest_dir"
    fi
  fi

  if [ -w "$dest_dir" ]; then
    install -m 755 "$src" "$dest"
  elif command -v sudo >/dev/null 2>&1; then
    info "Installing to ${dest} (may prompt for sudo)..."
    sudo install -m 755 "$src" "$dest"
  else
    die "install directory is not writable (${dest_dir}) and sudo is not available"
  fi

  printf '%s' "$dest"
}

path_contains_dir() {
  dir="$1"
  old_ifs=$IFS
  IFS=:
  for entry in $PATH; do
    if [ "$entry" = "$dir" ]; then
      IFS=$old_ifs
      return 0
    fi
  done
  IFS=$old_ifs
  return 1
}

is_sideguard_checkout() {
  dir="$1"
  [ -f "${dir}/go.mod" ] || return 1
  grep -q "module ${GO_MODULE}" "${dir}/go.mod" 2>/dev/null || return 1
  [ -d "${dir}/.git" ] || [ -f "${dir}/go.mod" ]
}

find_checkout_dir() {
  if [ -n "${script_path:-}" ]; then
    script_dir=$(CDPATH= cd -- "$(dirname -- "$script_path")" && pwd)
    if is_sideguard_checkout "$script_dir"; then
      printf '%s' "$script_dir"
      return 0
    fi
  fi

  cwd=$(pwd)
  if is_sideguard_checkout "$cwd"; then
    printf '%s' "$cwd"
    return 0
  fi

  return 1
}

resolve_install_mode() {
  if [ -n "${SIDEGUARD_INSTALL_MODE:-}" ]; then
    case "$SIDEGUARD_INSTALL_MODE" in
      github|source) printf '%s' "$SIDEGUARD_INSTALL_MODE"; return 0 ;;
      *)
        die "invalid SIDEGUARD_INSTALL_MODE: ${SIDEGUARD_INSTALL_MODE} (use github or source)"
        ;;
    esac
  fi

  if [ -t 0 ]; then
    prompt_install_mode
    return 0
  fi

  info "Non-interactive install: using GitHub download (set SIDEGUARD_INSTALL_MODE=source to build from source)."
  printf '%s' "github"
}

prompt_install_mode() {
  info ""
  info "How do you want to install SideGuard?"
  info "  1) Download pre-built binary from GitHub (recommended)"
  info "  2) Build from source (requires git, Go, CGO)"
  info ""
  while true; do
    printf '%s' "Enter 1 or 2 [1]: " >&2
    if ! read -r choice; then
      choice=1
    fi
    choice=${choice:-1}
    case "$choice" in
      1 | github | y | Y | yes | Yes)
        printf '%s' "github"
        return 0
        ;;
      2 | source | n | N | no | No)
        printf '%s' "source"
        return 0
        ;;
      *)
        info "Please enter 1 or 2."
        ;;
    esac
  done
}

require_cgo_toolchain() {
  need_cmd git
  need_cmd go

  if ! command -v cc >/dev/null 2>&1 \
    && ! command -v gcc >/dev/null 2>&1 \
    && ! command -v clang >/dev/null 2>&1; then
    die "CGO requires a C compiler (cc, gcc, or clang) — install Xcode CLT on macOS or build-essential on Linux"
  fi

  cgo_enabled=$(CGO_ENABLED=1 go env CGO_ENABLED 2>/dev/null || printf '0')
  if [ "$cgo_enabled" != "1" ]; then
    die "CGO is disabled in this Go toolchain; tray support requires CGO_ENABLED=1"
  fi
}

resolve_source_version() {
  repo_dir="$1"
  if [ -d "${repo_dir}/.git" ]; then
    version=$(git -C "$repo_dir" describe --tags --always --dirty 2>/dev/null || true)
    if [ -n "$version" ]; then
      printf '%s' "$version"
      return 0
    fi
  fi
  printf '%s' "dev"
}

post_install() {
  installed_path="$1"

  installed_version=$("$installed_path" --version 2>/dev/null || true)
  if [ -n "$installed_version" ]; then
    info "Installed: ${installed_version}"
  else
    info "Installed: ${installed_path}"
  fi

  if ! path_contains_dir "$install_dir"; then
    info ""
    info "Warning: ${install_dir} is not on your PATH."
    info "Add it to your shell profile, for example:"
    info "  export PATH=\"${install_dir}:\$PATH\""
  fi

  if [ "${SIDEGUARD_RUN_INSTALL:-0}" = "1" ]; then
    info ""
    info "Running sideguard install..."
    "$installed_path" install
  fi

  info ""
  info "SideGuard binary installed successfully."
  info ""
  info "Next steps:"
  info "  sideguard daemon start"
  info "  sideguard install          # wire Cursor/Claude hooks + MCP wrap + daemon"
  info "  sideguard status"
  info "  sideguard clients reload   # reload hooks/MCP in Cursor & Claude Code"
  info ""
  info "Optional: re-run with SIDEGUARD_RUN_INSTALL=1 to run sideguard install automatically."
  if [ "$os" = "linux" ]; then
    info ""
    info "Linux note: menu-bar tray and login auto-start differ from macOS."
    info "Use sideguard daemon start (or configure a user systemd unit) for the daemon."
  fi
  if [ "$os" = "darwin" ]; then
    info ""
    info "macOS note: unsigned release binaries may be blocked by Gatekeeper on first run."
    info "Right-click the binary -> Open, or: xattr -dr com.apple.quarantine ${installed_path}"
  fi
}

install_from_github() {
  need_cmd curl
  need_cmd tar
  need_cmd install

  requested_version="${SIDEGUARD_VERSION:-latest}"
  if [ "$requested_version" = "latest" ]; then
    info "Resolving latest ${GITHUB_OWNER}/${GITHUB_REPO} release..."
    tag=$(resolve_latest_tag)
  else
    tag=$(normalize_tag "$requested_version")
  fi

  version="${tag#v}"
  archive="sideguard_${version}_${os}_${arch}.tar.gz"
  base_url="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${tag}"

  info "Platform: ${os}/${arch}"
  info "Release: ${tag}"
  info "Archive: ${archive}"
  info "Install dir: ${install_dir}"

  tmpdir=$(mktemp -d 2>/dev/null || mktemp -d -t sideguard-install)
  trap 'rm -rf "$tmpdir"' EXIT INT HUP TERM

  info "Downloading checksums..."
  curl -fsSL -o "${tmpdir}/checksums.txt" "${base_url}/checksums.txt" \
    || die "failed to download checksums.txt for ${tag}"

  info "Downloading ${archive}..."
  curl -fsSL -o "${tmpdir}/${archive}" "${base_url}/${archive}" \
    || die "failed to download ${archive} (is this platform published for ${tag}?)"

  info "Verifying SHA256 checksum..."
  verify_checksum "$archive" "${tmpdir}/${archive}" "${tmpdir}/checksums.txt"

  info "Extracting archive..."
  tar -xzf "${tmpdir}/${archive}" -C "$tmpdir" \
    || die "failed to extract ${archive}"

  binary_path="${tmpdir}/sideguard"
  if [ ! -f "$binary_path" ]; then
    die "archive did not contain a sideguard binary"
  fi

  info "Installing binary..."
  installed_path=$(install_binary "$binary_path" "$install_dir")
  post_install "$installed_path"
}

install_from_source() {
  require_cgo_toolchain

  info "Platform: ${os}/${arch}"
  info "Install dir: ${install_dir}"

  cleanup_clone=0
  repo_dir=""
  build_dir=""

  if repo_dir=$(find_checkout_dir); then
    info "Using existing SideGuard checkout: ${repo_dir}"
    build_dir="${repo_dir}/bin"
    mkdir -p "$build_dir"
  else
    info "Cloning ${GITHUB_CLONE_URL}..."
    tmpdir=$(mktemp -d 2>/dev/null || mktemp -d -t sideguard-source)
    trap 'rm -rf "$tmpdir"' EXIT INT HUP TERM
    cleanup_clone=1
    repo_dir="${tmpdir}/sideguard"
    git clone --depth 1 "$GITHUB_CLONE_URL" "$repo_dir" \
      || die "failed to clone ${GITHUB_CLONE_URL}"
    build_dir="${repo_dir}/bin"
    mkdir -p "$build_dir"
  fi

  version=$(resolve_source_version "$repo_dir")
  binary_path="${build_dir}/sideguard"

  info "Building sideguard (CGO_ENABLED=1, version=${version})..."
  (
    cd "$repo_dir" || die "could not enter repository directory: ${repo_dir}"
    CGO_ENABLED=1 go build \
      -ldflags "-X ${GO_MODULE}/cmd/sideguard/cmd.Version=${version}" \
      -o "$binary_path" \
      ./cmd/sideguard
  ) || die "go build failed"

  if [ ! -f "$binary_path" ]; then
    die "build did not produce ${binary_path}"
  fi

  info "Installing binary..."
  installed_path=$(install_binary "$binary_path" "$install_dir")

  if [ "$cleanup_clone" -eq 1 ]; then
    rm -rf "$tmpdir"
    trap - EXIT INT HUP TERM
  fi

  post_install "$installed_path"
}

main() {
  if [ -f "$0" ] && [ "$0" != "sh" ] && [ "$0" != "bash" ] && [ "$0" != "-sh" ]; then
    script_path=$0
  fi

  detect_platform

  install_dir="${SIDEGUARD_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
  install_mode=$(resolve_install_mode)

  case "$install_mode" in
    github)
      install_from_github
      ;;
    source)
      install_from_source
      ;;
    *)
      die "unsupported install mode: ${install_mode}"
      ;;
  esac
}

main "$@"
