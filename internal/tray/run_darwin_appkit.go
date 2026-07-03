// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

//go:build darwin

package tray

import (
	"runtime"

	"github.com/alisaitteke/sideguard/internal/tray/darwin"
)

// darwinPrepare configures NSApplication on the locked OS thread (mirrors systray Register).
func darwinPrepare() {
	runtime.LockOSThread()
	darwin.Prepare()
}

// runDarwinAppKitLoop blocks in [NSApp run] on the locked OS thread (mirrors systray nativeLoop).
func runDarwinAppKitLoop() {
	runtime.LockOSThread()
	darwin.RunLoop()
}
