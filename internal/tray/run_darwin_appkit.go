//go:build darwin

package tray

import (
	"runtime"

	"github.com/alisaitteke/vibeguard/internal/tray/darwin"
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
