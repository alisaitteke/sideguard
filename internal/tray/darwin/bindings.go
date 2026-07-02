//go:build darwin

// Package darwin provides the macOS AppKit tray shell (NSStatusItem + NSPopover).
// See docs/plans/2026-07-01-1537-mac-tray-popover/ (mtp-phase-3.0-darwin-panel.md).
package darwin

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#include "status_popover.h"

extern void goTrayReady(void);

static void tray_ready_bridge(void) {
	goTrayReady();
}

static void darwin_tray_prepare_bridge(void) {
	darwin_tray_prepare(tray_ready_bridge);
}

static void darwin_tray_run_loop_bridge(void) {
	darwin_tray_run_loop();
}
*/
import "C"
import (
	"encoding/json"
	"runtime"
	"unsafe"
)

func init() {
	// AppKit requires [NSApp run] on the process main thread (mirrors getlantern/systray).
	runtime.LockOSThread()
}

var (
	trayReady       func()
	decideHandler   func(id, decision string)
	modeHandler     func(modeIndex int)
	quitHandler     func()
	installHandler  func()
)

// PanelJSONRow is one Allow/Deny row in the popover JSON payload.
type PanelJSONRow struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// PanelJSON is the ObjC bridge payload for darwin_update_panel.
type PanelJSON struct {
	DaemonStatus  string         `json:"daemon_status"`
	PendingCount  string         `json:"pending_count"`
	ModeIndex     int            `json:"mode_index"`
	ModeEnabled   bool           `json:"mode_enabled"`
	Rows          []PanelJSONRow `json:"rows"`
	OverflowHint  string         `json:"overflow_hint"`
	EmptyMessage  string         `json:"empty_message"`
	UpdateVisible bool           `json:"update_visible"`
	UpdateLabel   string         `json:"update_label"`
	UpdateEnabled bool           `json:"update_enabled"`
}

// SetReadyHandler registers the callback invoked on the main thread once AppKit setup completes.
// Must be called before Run.
func SetReadyHandler(fn func()) {
	trayReady = fn
}

// SetDecideHandler registers the Allow/Deny callback from panel buttons.
func SetDecideHandler(fn func(id, decision string)) {
	decideHandler = fn
}

// SetModeHandler registers the segmented-control mode change callback.
func SetModeHandler(fn func(modeIndex int)) {
	modeHandler = fn
}

// SetQuitHandler registers the popover Quit button callback.
func SetQuitHandler(fn func()) {
	quitHandler = fn
}

// SetInstallHandler registers the popover Install update button callback.
func SetInstallHandler(fn func()) {
	installHandler = fn
}

//export goTrayReady
func goTrayReady() {
	if trayReady == nil {
		return
	}
	// Mirror getlantern/systray: unblock a worker goroutine; do not run tray setup on
	// the AppKit main thread during applicationDidFinishLaunching.
	trayReady()
}

//export goDecide
func goDecide(cID, cDecision *C.char) {
	if decideHandler == nil {
		return
	}
	decideHandler(C.GoString(cID), C.GoString(cDecision))
}

//export goSetMode
func goSetMode(modeIndex C.int) {
	if modeHandler == nil {
		return
	}
	modeHandler(int(modeIndex))
}

//export goQuitTray
func goQuitTray() {
	if quitHandler != nil {
		quitHandler()
	}
}

//export goInstallUpdate
func goInstallUpdate() {
	if installHandler != nil {
		installHandler()
	}
}

// Prepare configures NSApplication and the status-item delegate. Call from a different
// Go function than RunLoop (mirrors getlantern/systray registerSystray vs nativeLoop).
func Prepare() {
	runtime.LockOSThread()
	C.darwin_tray_prepare_bridge()
}

// RunLoop blocks in [NSApp run] until Quit. Must follow Prepare on the locked OS thread.
func RunLoop() {
	runtime.LockOSThread()
	C.darwin_tray_run_loop_bridge()
}

// SetIcon updates the menu-bar icon from PNG bytes.
func SetIcon(data []byte) {
	if len(data) == 0 {
		return
	}
	C.darwin_set_icon((*C.uchar)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
}

// SetTitle sets the optional menu-bar title (pending count).
func SetTitle(title string) {
	cstr := C.CString(title)
	defer C.free(unsafe.Pointer(cstr))
	C.darwin_set_title(cstr)
}

// SetTooltip sets the status-item tooltip.
func SetTooltip(tooltip string) {
	cstr := C.CString(tooltip)
	defer C.free(unsafe.Pointer(cstr))
	C.darwin_set_tooltip(cstr)
}

// ShowPopover opens the popover below the menu-bar icon if it is currently hidden.
// No-op when already visible (avoids flicker on periodic poll updates).
func ShowPopover() {
	C.darwin_show_popover()
}

// UpdatePanel rebuilds popover content from a JSON snapshot on the main thread.
func UpdatePanel(payload PanelJSON) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	cstr := C.CString(string(data))
	defer C.free(unsafe.Pointer(cstr))
	C.darwin_update_panel(cstr)
}

// Quit terminates the NSApplication run loop.
func Quit() {
	C.darwin_quit()
}
