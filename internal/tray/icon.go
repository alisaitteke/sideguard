package tray

// menuBarIcon returns the default (idle, healthy, no pending) tray icon PNG bytes at 32 px.
func menuBarIcon() []byte {
	return mustMenuBarIconPNG(0, true, menuBarIconSize)
}

// menuBarIconForState picks tray icon PNG bytes from the latest poll snapshot.
// Pending count is drawn inside the badge circle when the daemon is healthy and
// at least one approval waits; otherwise logo-check.svg is shown.
func menuBarIconForState(pending int, healthOK bool) []byte {
	return mustMenuBarIconPNG(pending, healthOK, menuBarIconSize)
}

// menuBarIconForStateAtSize renders the icon at an explicit size (16 or 32 px).
func menuBarIconForStateAtSize(pending int, healthOK bool, size int) []byte {
	return mustMenuBarIconPNG(pending, healthOK, size)
}

func mustMenuBarIconPNG(pending int, healthOK bool, size int) []byte {
	data, err := renderMenuBarIconPNG(pending, healthOK, size)
	if err != nil {
		panic("tray: menu bar icon render failed: " + err.Error())
	}
	return data
}

// popoverHeaderLogo returns the branded check-mark logo for the popover header.
// When dark is true, opaque pixels are rendered white for dark popover backgrounds.
func popoverHeaderLogo(dark bool) []byte {
	data, err := renderPopoverHeaderLogoPNG(dark)
	if err != nil {
		panic("tray: popover header logo render failed: " + err.Error())
	}
	return data
}
