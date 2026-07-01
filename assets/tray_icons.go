// Package assets holds embedded brand assets for VibeGuard.
package assets

import _ "embed"

// Menu-bar tray SVG sources (512×512 viewBox). Rasterized at 16/32 px in internal/tray.
//
//go:embed logo-check.svg
var LogoCheckSVG []byte

//go:embed logo-empty.svg
var LogoEmptySVG []byte
