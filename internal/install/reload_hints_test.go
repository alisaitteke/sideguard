package install_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/install"
)

func capturePrint(fn func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestPrintClientReloadHints_CursorOnly(t *testing.T) {
	out := capturePrint(func() {
		install.PrintClientReloadHints(install.Options{Cursor: true}, "install changes", install.ReloadHintsBrief)
	})
	if strings.Contains(out, "Claude Code:") {
		t.Fatal("expected Claude section omitted")
	}
	if !strings.Contains(out, "Reload Window") {
		t.Fatal("expected Cursor reload window hint")
	}
	if !strings.Contains(out, "vibeguard clients reload") {
		t.Fatal("expected pointer to clients reload command")
	}
}

func TestPrintClientReloadHints_FullClaude(t *testing.T) {
	out := capturePrint(func() {
		install.PrintClientReloadHints(install.Options{Claude: true}, "config changes", install.ReloadHintsFull)
	})
	if strings.Contains(out, "Cursor:") {
		t.Fatal("expected Cursor section omitted")
	}
	if !strings.Contains(out, "/exit") {
		t.Fatal("expected Claude session restart hint")
	}
	if !strings.Contains(out, "/hooks lists") {
		t.Fatal("expected /hooks clarification")
	}
	if strings.Contains(out, "vibeguard clients reload") {
		t.Fatal("full verbosity should not point to clients reload")
	}
}

func TestPrintClientReloadHints_DefaultBothClients(t *testing.T) {
	var buf bytes.Buffer
	// default flags false,false should still print both when called like install with no flags
	out := capturePrint(func() {
		install.PrintClientReloadHints(install.Options{}, "changes", install.ReloadHintsBrief)
	})
	_ = buf
	if !strings.Contains(out, "Cursor:") || !strings.Contains(out, "Claude Code:") {
		t.Fatalf("expected both clients, got:\n%s", out)
	}
}
