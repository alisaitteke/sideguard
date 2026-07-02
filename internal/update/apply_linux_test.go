//go:build linux

package update

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHasSystemdDaemonUnit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	unitPath := filepath.Join(unitDir, systemdDaemonUnit)
	if err := os.WriteFile(unitPath, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasSystemdDaemonUnit() {
		t.Fatal("expected systemd unit to exist")
	}
}

func TestLinuxApplierStopBestEffort(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	applier := linuxApplier{}
	if err := applier.Stop(context.Background()); err != nil {
		t.Fatalf("Stop should be best-effort: %v", err)
	}
}
