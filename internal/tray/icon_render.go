package tray

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sync"

	"github.com/alisaitteke/vibeguard/assets"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	// menuBarIconSize is the primary raster size for macOS menu-bar icons (32 px @2x → 18 pt).
	menuBarIconSize = 32
	// menuBarIconSize1x is the 1× companion size for completeness and tests.
	menuBarIconSize1x = 16

	// countCenterX and countCenterY locate the pending-count overlay in normalized
	// viewBox coordinates (512×512). Anchored to the green check badge in logo-check.svg.
	countCenterX = 368.0 / 512.0
	countCenterY = 340.0 / 512.0
)

var iconPNGCache sync.Map // string → []byte

// rasterizeSVG renders embedded SVG bytes to an RGBA image at the given square size.
func rasterizeSVG(svgData []byte, size int) (*image.RGBA, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgData))
	if err != nil {
		return nil, fmt.Errorf("parse svg: %w", err)
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(rgba, rgba.Bounds(), image.Transparent, image.Point{}, draw.Src)
	scanner := rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1.0)
	return rgba, nil
}

// drawPendingCount overlays the pending approval count inside the shield badge circle.
func drawPendingCount(img *image.RGBA, count int, size int) {
	if count <= 0 {
		return
	}
	label := fmt.Sprintf("%d", count)
	if count > 99 {
		label = "99+"
	}

	ttf, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return
	}
	fontSize := float64(size) * 0.30
	if len(label) > 1 {
		fontSize = float64(size) * 0.22
	}
	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return
	}
	defer face.Close()

	cx := int(countCenterX * float64(size))
	cy := int(countCenterY * float64(size))

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{0x22, 0xad, 0x00, 0xff}),
		Face: face,
	}
	bounds, _ := d.BoundString(label)
	width := (bounds.Max.X - bounds.Min.X).Ceil()
	height := (bounds.Max.Y - bounds.Min.Y).Ceil()
	d.Dot = fixed.P(
		cx-width/2,
		cy+height/2-bounds.Min.Y.Ceil(),
	)
	d.DrawString(label)
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderMenuBarIconPNG(pending int, healthOK bool, size int) ([]byte, error) {
	key := menuBarIconCacheKey(pending, healthOK, size)
	if cached, ok := iconPNGCache.Load(key); ok {
		return cached.([]byte), nil
	}

	var svgData []byte
	switch {
	case !healthOK || pending == 0:
		svgData = assets.LogoCheckSVG
	default:
		svgData = assets.LogoEmptySVG
	}

	img, err := rasterizeSVG(svgData, size)
	if err != nil {
		return nil, err
	}
	if healthOK && pending > 0 {
		drawPendingCount(img, pending, size)
	}

	pngBytes, err := encodePNG(img)
	if err != nil {
		return nil, err
	}
	iconPNGCache.Store(key, pngBytes)
	return pngBytes, nil
}

func menuBarIconCacheKey(pending int, healthOK bool, size int) string {
	if !healthOK || pending == 0 {
		return fmt.Sprintf("check:%d", size)
	}
	return fmt.Sprintf("pending:%d:%d", pending, size)
}
