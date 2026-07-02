package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
)

func TestNextModeCycles(t *testing.T) {
	t.Parallel()
	if got := nextMode(approvalmode.Ask); got != approvalmode.Auto {
		t.Fatalf("ask -> %q", got)
	}
	if got := nextMode(approvalmode.Auto); got != approvalmode.AutoAllow {
		t.Fatalf("auto -> %q", got)
	}
	if got := nextMode(approvalmode.AutoAllow); got != approvalmode.AutoDeny {
		t.Fatalf("auto_allow -> %q", got)
	}
	if got := nextMode(approvalmode.AutoDeny); got != approvalmode.Ask {
		t.Fatalf("auto_deny -> %q", got)
	}
}

func TestModeBanner(t *testing.T) {
	t.Parallel()
	if modeBanner(approvalmode.Ask) != "" {
		t.Fatal("ask should have no banner")
	}
	if modeBanner(approvalmode.Auto) != "" {
		t.Fatal("auto (smart triage) should have no banner")
	}
	if !strings.Contains(modeBanner(approvalmode.AutoAllow), "AUTO-ALLOW") {
		t.Fatal("auto-allow banner missing")
	}
	if !strings.Contains(modeBanner(approvalmode.AutoDeny), "AUTO-DENY") {
		t.Fatal("auto-deny banner missing")
	}
}

func TestUpdateRefreshSetsMode(t *testing.T) {
	t.Parallel()

	m := newModel(nil)
	updated, cmd := m.Update(refreshDoneMsg{
		items: []api.PendingApproval{{ID: "abc-123", Command: "git status"}},
		mode:  approvalmode.AutoAllow,
	})
	if cmd != nil {
		t.Fatal("expected no command after refresh")
	}
	um := updated.(model)
	if um.mode != approvalmode.AutoAllow {
		t.Fatalf("mode = %q", um.mode)
	}
}

func TestUpdateGKeyQueuesSetMode(t *testing.T) {
	t.Parallel()

	m := newModel(nil)
	m.mode = approvalmode.Ask
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd == nil {
		t.Fatal("expected set mode command")
	}
	um := updated.(model)
	if !strings.Contains(um.flash, "Auto") {
		t.Fatalf("flash = %q", um.flash)
	}
}

func TestViewAutoModeBanner(t *testing.T) {
	t.Parallel()

	m := newModel(nil)
	m.mode = approvalmode.AutoDeny
	view := m.View()
	if !strings.Contains(view, "AUTO-DENY MODE") {
		t.Fatalf("view missing banner: %q", view)
	}
	if !strings.Contains(view, "Mode: Auto-deny") {
		t.Fatalf("view missing footer mode: %q", view)
	}
}

func TestSetModeDoneUpdatesMode(t *testing.T) {
	t.Parallel()

	m := newModel(nil)
	updated, cmd := m.Update(setModeDoneMsg{mode: approvalmode.AutoAllow})
	um := updated.(model)
	if um.mode != approvalmode.AutoAllow {
		t.Fatalf("mode = %q", um.mode)
	}
	if cmd == nil {
		t.Fatal("expected refresh after set mode")
	}
}
