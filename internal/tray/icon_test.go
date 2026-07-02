package tray

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestMenuBarIcon_nonEmpty(t *testing.T) {
	if len(menuBarIcon()) == 0 {
		t.Fatal("menuBarIcon() returned empty bytes")
	}
	if _, err := png.Decode(bytes.NewReader(menuBarIcon())); err != nil {
		t.Fatalf("menuBarIcon() is not valid PNG: %v", err)
	}
}

func TestMenuBarIconForState(t *testing.T) {
	t.Parallel()

	idle := menuBarIconForState(0, true)
	if len(idle) == 0 {
		t.Fatal("idle healthy state returned empty icon")
	}
	if _, err := png.Decode(bytes.NewReader(idle)); err != nil {
		t.Fatalf("idle icon is not valid PNG: %v", err)
	}

	down := menuBarIconForState(3, false)
	if string(down) != string(idle) {
		t.Fatal("daemon down should use check icon regardless of pending count")
	}

	pending := menuBarIconForState(3, true)
	if len(pending) == 0 {
		t.Fatal("healthy pending state returned empty icon")
	}
	if string(pending) == string(idle) {
		t.Fatal("pending icon must differ from idle check icon")
	}
	if _, err := png.Decode(bytes.NewReader(pending)); err != nil {
		t.Fatalf("pending icon is not valid PNG: %v", err)
	}
}

func TestRenderMenuBarIconPNG_sizes(t *testing.T) {
	t.Parallel()

	for _, size := range []int{menuBarIconSize1x, menuBarIconSize} {
		data, err := renderMenuBarIconPNG(0, true, size)
		if err != nil {
			t.Fatalf("size %d check render: %v", size, err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("size %d check PNG decode: %v", size, err)
		}
		b := img.Bounds()
		if b.Dx() != size || b.Dy() != size {
			t.Fatalf("size %d: got %dx%d image", size, b.Dx(), b.Dy())
		}
	}
}

func TestRenderMenuBarIconPNG_pendingCountOverlay(t *testing.T) {
	t.Parallel()

	one, err := renderMenuBarIconPNG(1, true, menuBarIconSize)
	if err != nil {
		t.Fatal(err)
	}
	three, err := renderMenuBarIconPNG(3, true, menuBarIconSize)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(one, three) {
		t.Fatal("icons for pending 1 and 3 should differ (count overlay)")
	}
}

func TestRenderPopoverHeaderLogoPNG_appearanceVariants(t *testing.T) {
	t.Parallel()

	light, err := renderPopoverHeaderLogoPNG(false)
	if err != nil {
		t.Fatal(err)
	}
	dark, err := renderPopoverHeaderLogoPNG(true)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(light, dark) {
		t.Fatal("light and dark header logos should differ")
	}

	lightImg, err := png.Decode(bytes.NewReader(light))
	if err != nil {
		t.Fatalf("light logo PNG decode: %v", err)
	}
	darkImg, err := png.Decode(bytes.NewReader(dark))
	if err != nil {
		t.Fatalf("dark logo PNG decode: %v", err)
	}

	if hasOpaqueNonWhitePixel(darkImg) {
		t.Fatal("dark header logo should render opaque pixels as white")
	}
	if !hasOpaqueNonWhitePixel(lightImg) {
		t.Fatal("light header logo should keep brand fill (non-white opaque pixels)")
	}
}

func hasOpaqueNonWhitePixel(img image.Image) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			if r>>8 < 250 || g>>8 < 250 || b>>8 < 250 {
				return true
			}
		}
	}
	return false
}
