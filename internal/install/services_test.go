package install_test

import (
	"runtime"
	"testing"

	"github.com/alisaitteke/sideguard/internal/install"
)

func TestShouldInstallTray(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("tray LaunchAgent is macOS-only")
	}

	if install.ShouldInstallTray(install.Options{}) != true {
		t.Fatal("expected tray install by default on darwin")
	}
	if install.ShouldInstallTray(install.Options{Headless: true}) {
		t.Fatal("expected headless to skip tray")
	}
}
