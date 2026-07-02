//go:build darwin || linux

package update

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicSwapBinary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sideguard")
	staging := filepath.Join(dir, "staging")
	newContent := []byte("#!/bin/sh\necho v2\n")
	oldContent := []byte("#!/bin/sh\necho v1\n")

	if err := os.WriteFile(target, oldContent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staging, newContent, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := atomicSwapBinary(staging, target); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, target); string(got) != string(newContent) {
		t.Fatalf("target = %q, want %q", got, newContent)
	}
	if _, err := os.Stat(target + ".new"); err == nil {
		t.Fatal("expected .new file removed after swap")
	}
}

func TestNoopPlatformApplierSwap(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sideguard")
	staging := filepath.Join(dir, "staging")
	payload := []byte("noop-swap")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staging, payload, 0o755); err != nil {
		t.Fatal(err)
	}

	applier := NoopPlatformApplier{}
	if err := applier.SwapBinary(context.Background(), staging, target); err != nil {
		t.Fatal(err)
	}
	if string(readFile(t, target)) != string(payload) {
		t.Fatal("swap via noop applier failed")
	}
}

func TestNewPlatformApplierImplementsInterface(t *testing.T) {
	applier := NewPlatformApplier()
	if applier == nil {
		t.Fatal("expected non-nil platform applier")
	}
	// Best-effort stop when nothing is running must not fail the apply path.
	if err := applier.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
