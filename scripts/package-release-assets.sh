#!/usr/bin/env bash
# Packages GoReleaser build outputs into release archives and checksums.txt.
# Consumed by .github/workflows/release.yml merge job.
# See docs/plans/2026-07-02-1102-github-update/vgu-phase-1.0-release-ci.md
set -euo pipefail

tag="${1:?usage: package-release-assets.sh <tag> [dist-dir]}"
dist_root="${2:-dist}"
version="${tag#v}"
out_dir="${dist_root}/release"

rm -rf "${out_dir}"
mkdir -p "${out_dir}"
out_dir_abs="$(cd "${out_dir}" && pwd)"

shopt -s nullglob
for dir in "${dist_root}"/*_*; do
  [[ -d "${dir}" ]] || continue
  base="$(basename "${dir}")"
  # GoReleaser v2 build output: {build_id}_{goos}_{goarch}_v{variant}
  if [[ ! "${base}" =~ ^(darwin|linux|windows)_ ]]; then
    continue
  fi
  target="${base%_v*}"
  target="${target#*_}"

  archive_base="vibeguard_${version}_${target}"
  if [[ "${target}" == windows_* ]]; then
    (cd "${dir}" && zip -q "${out_dir_abs}/${archive_base}.zip" vibeguard.exe)
  else
    tar -czf "${out_dir}/${archive_base}.tar.gz" -C "${dir}" vibeguard
  fi
done

if ! compgen -G "${out_dir}/vibeguard_*" > /dev/null; then
  echo "package-release-assets: no archives created under ${dist_root}" >&2
  exit 1
fi

(
  cd "${out_dir}"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum vibeguard_* > checksums.txt
  else
    shasum -a 256 vibeguard_* > checksums.txt
  fi
)

echo "Packaged release assets in ${out_dir}:"
ls -la "${out_dir}"
