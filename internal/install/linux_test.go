//go:build linux

package install

import (
	"strings"
	"testing"
)

func TestRenderSystemdUnit(t *testing.T) {
	content, err := RenderSystemdUnit("/usr/local/bin/sideguard")
	if err != nil {
		t.Fatalf("RenderSystemdUnit: %v", err)
	}
	if !strings.Contains(content, "ExecStart=/usr/local/bin/sideguard daemon run") {
		t.Fatalf("missing ExecStart: %s", content)
	}
	if !strings.Contains(content, "[Install]") {
		t.Fatalf("missing Install section: %s", content)
	}
}
