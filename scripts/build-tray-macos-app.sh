#!/usr/bin/env bash
# Builds dist/VibeGuard Tray.app — macOS menu-bar bundle with LSUIElement (no Dock icon).
# See docs/plans/2026-07-01-1355-go-systray-tray/ (gst-phase-4.0-macos-packaging.md).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="${BINARY:-$ROOT/bin/vibeguard}"
APP_NAME="VibeGuard Tray"
DIST="$ROOT/dist"
APP="$DIST/$APP_NAME.app"
ICON_SRC="$ROOT/internal/tray/assets/icon_32.png"
ICNS_ASSET="$ROOT/assets/tray/AppIcon.icns"

if [[ "$(uname -s)" != "Darwin" ]]; then
	echo "tray-app build requires macOS (CGO systray .app bundle)" >&2
	exit 1
fi

if [[ ! -x "$BINARY" ]]; then
	echo "binary not found: $BINARY — run 'CGO_ENABLED=1 make build' first" >&2
	exit 1
fi

if [[ ! -f "$ICON_SRC" ]]; then
	echo "icon source not found: $ICON_SRC" >&2
	exit 1
fi

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

cp "$BINARY" "$APP/Contents/MacOS/vibeguard"
chmod +x "$APP/Contents/MacOS/vibeguard"

cat >"$APP/Contents/MacOS/vibeguard-tray" <<'EOF'
#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
exec "$DIR/vibeguard" tray
EOF
chmod +x "$APP/Contents/MacOS/vibeguard-tray"

ICNS_DST="$APP/Contents/Resources/AppIcon.icns"
if [[ -f "$ICNS_ASSET" ]]; then
	cp "$ICNS_ASSET" "$ICNS_DST"
else
	ICONSET="$ROOT/assets/tray/.build-iconset.iconset"
	rm -rf "$ICONSET"
	mkdir -p "$ICONSET"
	trap 'rm -rf "$ICONSET"' EXIT
	make_icon() {
		local size=$1
		local name=$2
		sips -z "$size" "$size" "$ICON_SRC" --out "$ICONSET/${name}.png" >/dev/null
	}
	make_icon 16 icon_16x16
	make_icon 32 icon_16x16@2x
	make_icon 32 icon_32x32
	make_icon 64 icon_32x32@2x
	make_icon 128 icon_128x128
	make_icon 256 icon_128x128@2x
	make_icon 256 icon_256x256
	make_icon 512 icon_256x256@2x
	make_icon 512 icon_512x512
	make_icon 1024 icon_512x512@2x
	iconutil -c icns "$ICONSET" -o "$ICNS_DST"
	mkdir -p "$(dirname "$ICNS_ASSET")"
	cp "$ICNS_DST" "$ICNS_ASSET"
fi

cat >"$APP/Contents/Info.plist" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundleExecutable</key>
	<string>vibeguard-tray</string>
	<key>CFBundleIconFile</key>
	<string>AppIcon</string>
	<key>CFBundleIdentifier</key>
	<string>com.vibeguard.tray</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>VibeGuard Tray</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0</string>
	<key>CFBundleVersion</key>
	<string>1</string>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
	<key>LSUIElement</key>
	<true/>
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
EOF

echo "Built $APP"
