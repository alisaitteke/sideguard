package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/alisaitteke/vibeguard/internal/api"
)

func TestPendingIDsForAutoApprove(t *testing.T) {
	t.Parallel()

	items := []api.PendingApproval{
		{ID: "first"},
		{ID: "second"},
		{ID: "third"},
	}

	got := pendingIDsForAutoApprove(items, map[string]bool{"second": true})
	want := []string{"first", "third"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPendingIDsForAutoApproveEmpty(t *testing.T) {
	t.Parallel()
	if got := pendingIDsForAutoApprove(nil, nil); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestNewModelAutoApprove(t *testing.T) {
	t.Parallel()

	m := newModel(nil, Options{AutoApprove: true})
	if !m.autoApprove {
		t.Fatal("expected autoApprove true")
	}
	if m.deciding == nil {
		t.Fatal("expected deciding map initialized")
	}
}

func TestUpdateRefreshAutoApproveQueuesDecisions(t *testing.T) {
	t.Parallel()

	m := newModel(nil, Options{AutoApprove: true})
	updated, cmd := m.Update(refreshDoneMsg{
		items: []api.PendingApproval{
			{ID: "abc-123", Command: "git status"},
		},
	})
	if cmd == nil {
		t.Fatal("expected decide command batch")
	}

	um := updated.(model)
	if !um.deciding["abc-123"] {
		t.Fatal("expected deciding flag for abc-123")
	}
}

func TestUpdateRefreshManualModeNoAutoDecide(t *testing.T) {
	t.Parallel()

	m := newModel(nil, Options{})
	_, cmd := m.Update(refreshDoneMsg{
		items: []api.PendingApproval{{ID: "abc-123", Command: "git status"}},
	})
	if cmd != nil {
		t.Fatal("expected no decide command without auto-approve")
	}
}

func TestUpdateDecideDoneAutoApproveFlash(t *testing.T) {
	t.Parallel()

	m := newModel(nil, Options{AutoApprove: true})
	m.items = []api.PendingApproval{{ID: "abc-123", Command: "git status"}}
	m.deciding = map[string]bool{"abc-123": true}

	updated, _ := m.Update(decideDoneMsg{decision: "allow", id: "abc-123"})
	um := updated.(model)
	if !strings.Contains(um.flash, "Auto-approved") {
		t.Fatalf("flash = %q, want Auto-approved prefix", um.flash)
	}
	if !strings.Contains(um.flash, "git status") {
		t.Fatalf("flash = %q, want command summary", um.flash)
	}
	if um.deciding["abc-123"] {
		t.Fatal("expected deciding flag cleared")
	}
}

func TestUpdateToggleAutoApproveWithG(t *testing.T) {
	t.Parallel()

	m := newModel(nil, Options{})
	m.items = []api.PendingApproval{{ID: "abc-123", Command: "git status"}}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	um := updated.(model)
	if !um.autoApprove {
		t.Fatal("expected autoApprove enabled after g")
	}
	if um.flash != "Auto-approve ON" {
		t.Fatalf("flash = %q, want Auto-approve ON", um.flash)
	}
	if cmd == nil {
		t.Fatal("expected decide command when enabling with pending items")
	}
	if !um.deciding["abc-123"] {
		t.Fatal("expected deciding flag for pending item")
	}

	updated2, cmd2 := um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	um2 := updated2.(model)
	if um2.autoApprove {
		t.Fatal("expected autoApprove disabled after second g")
	}
	if um2.flash != "Auto-approve OFF" {
		t.Fatalf("flash = %q, want Auto-approve OFF", um2.flash)
	}
	if cmd2 == nil {
		t.Fatal("expected flash clear command")
	}
}

func TestViewAutoApproveBanner(t *testing.T) {
	t.Parallel()

	m := newModel(nil, Options{AutoApprove: true})
	view := m.View()
	if !strings.Contains(view, "AUTO-APPROVE MODE") {
		t.Fatalf("view missing banner: %q", view)
	}
	if !strings.Contains(view, "Auto-approve OFF") {
		t.Fatalf("view missing toggle hint when on: %q", view)
	}
}

func TestFormatAutoApproveFlash(t *testing.T) {
	t.Parallel()

	flash := formatAutoApproveFlash("abc-123", []api.PendingApproval{
		{ID: "abc-123", Command: "npm test"},
	})
	if flash != "Auto-approved #abc-123 · npm test" {
		t.Fatalf("got %q", flash)
	}
}
