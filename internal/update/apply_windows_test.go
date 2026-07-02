//go:build windows

package update

import (
	"strings"
	"testing"
)

func TestRenderWindowsApplyHelper(t *testing.T) {
	target := `C:\Users\jo\.vibeguard\bin\vibeguard.exe`
	newPath := target + ".new"
	helper := `C:\Users\jo\.vibeguard\run\update\apply-helper.cmd`

	script := renderWindowsApplyHelper(target, newPath, helper)
	if !strings.Contains(script, target) {
		t.Fatalf("script missing target: %s", script)
	}
	if !strings.Contains(script, newPath) {
		t.Fatalf("script missing new path: %s", script)
	}
	if !strings.Contains(script, "daemon run") {
		t.Fatal("script should restart daemon")
	}
}

func TestNewPlatformApplierWindows(t *testing.T) {
	applier := NewPlatformApplier()
	if _, ok := applier.(windowsApplier); !ok {
		t.Fatalf("expected windowsApplier, got %T", applier)
	}
}
