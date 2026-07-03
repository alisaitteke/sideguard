// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

//go:build darwin

// Package darwin provides the macOS AppKit tray shell (NSStatusItem + NSPopover).
// See docs/plans/2026-07-02-1226-tray-ui-polish/ (tup-phase-3.0-tray-darwin.md).
package darwin

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#include "status_popover.h"

extern void goTrayReady(void);
extern void goAppearanceChanged(int dark);
extern void goAnalyseCommand(char *row_id, char *command, int use_event_id);

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
	trayReady            func()
	appearanceHandler    func(dark bool)
	decideHandler        func(id, decision string)
	modeHandler          func(modeIndex int)
	quitHandler          func()
	installHandler       func()
	loadMoreHandler      func()
	openSettingsHandler  func()
	saveSettingsHandler  func(payload string) error
	analyseHandler       func(rowID, command string, useEventID bool) AnalyseResultJSON
)

// PanelJSONRow is one pending or history row in the popover JSON payload.
type PanelJSONRow struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

// PanelJSON is the ObjC bridge payload for darwin_update_panel.
type PanelJSON struct {
	ModeIndex       int            `json:"mode_index"`
	ModeEnabled     bool           `json:"mode_enabled"`
	PendingRows     []PanelJSONRow `json:"pending_rows"`
	HistoryRows     []PanelJSONRow `json:"history_rows"`
	HistoryHasMore  bool           `json:"history_has_more"`
	PendingOverflow string         `json:"pending_overflow"`
	EmptyMessage    string         `json:"empty_message"`
	FooterDaemon    string         `json:"footer_daemon"`
	FooterPending   string         `json:"footer_pending"`
	UpdateVisible   bool           `json:"update_visible"`
	UpdateLabel     string         `json:"update_label"`
	UpdateEnabled   bool           `json:"update_enabled"`
}

// SetReadyHandler registers the callback invoked on the main thread once AppKit setup completes.
// Must be called before Run.
func SetReadyHandler(fn func()) {
	trayReady = fn
}

// SetAppearanceHandler registers the callback invoked when the popover effective appearance changes.
func SetAppearanceHandler(fn func(dark bool)) {
	appearanceHandler = fn
}

// SetDecideHandler registers the Run/Decline callback from panel buttons.
func SetDecideHandler(fn func(id, decision string)) {
	decideHandler = fn
}

// SetModeHandler registers the hamburger mode menu change callback.
func SetModeHandler(fn func(modeIndex int)) {
	modeHandler = fn
}

// SetQuitHandler registers the popover Quit confirmation callback.
func SetQuitHandler(fn func()) {
	quitHandler = fn
}

// SetInstallHandler registers the popover Install update button callback.
func SetInstallHandler(fn func()) {
	installHandler = fn
}

// SetLoadMoreHandler registers the explicit "Load more" button callback.
func SetLoadMoreHandler(fn func()) {
	loadMoreHandler = fn
}

// SetOpenSettingsHandler registers the hamburger Settings menu callback.
func SetOpenSettingsHandler(fn func()) {
	openSettingsHandler = fn
}

// SetSaveSettingsHandler registers the settings Save button callback.
func SetSaveSettingsHandler(fn func(payload string) error) {
	saveSettingsHandler = fn
}

// SetAnalyseHandler registers the detail-view Analyse button callback.
func SetAnalyseHandler(fn func(rowID, command string, useEventID bool) AnalyseResultJSON) {
	analyseHandler = fn
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

//export goAppearanceChanged
func goAppearanceChanged(dark C.int) {
	if appearanceHandler == nil {
		return
	}
	appearanceHandler(dark != 0)
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

//export goLoadMoreHistory
func goLoadMoreHistory() {
	if loadMoreHandler != nil {
		loadMoreHandler()
	}
}

//export goOpenSettings
func goOpenSettings() {
	if openSettingsHandler != nil {
		openSettingsHandler()
	}
}

//export goSaveSettings
func goSaveSettings(cJSON *C.char) {
	if saveSettingsHandler == nil {
		return
	}
	if err := saveSettingsHandler(C.GoString(cJSON)); err != nil {
		msg := C.CString(err.Error())
		defer C.free(unsafe.Pointer(msg))
		C.darwin_settings_show_error(msg)
	}
}

//export goAnalyseCommand
func goAnalyseCommand(cRowID, cCommand *C.char, useEventID C.int) {
	if analyseHandler == nil {
		return
	}
	rowID := C.GoString(cRowID)
	command := C.GoString(cCommand)
	useEv := useEventID != 0
	go func() {
		result := analyseHandler(rowID, command, useEv)
		data, err := json.Marshal(result)
		if err != nil {
			return
		}
		cstr := C.CString(string(data))
		defer C.free(unsafe.Pointer(cstr))
		C.darwin_update_analyse_result(cstr)
	}()
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

// SetHeaderLogo updates the popover header logo from PNG bytes.
func SetHeaderLogo(data []byte) {
	if len(data) == 0 {
		return
	}
	C.darwin_set_header_logo((*C.uchar)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
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

// ShowSettings opens the settings screen and populates it from a JSON snapshot.
func ShowSettings(snap SettingsJSON) {
	data, err := json.Marshal(snap)
	if err != nil {
		return
	}
	cstr := C.CString(string(data))
	defer C.free(unsafe.Pointer(cstr))
	C.darwin_show_settings(cstr)
}

// HideSettings returns from the settings screen to the list view.
func HideSettings() {
	C.darwin_hide_settings()
}
