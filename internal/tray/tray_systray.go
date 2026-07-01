//go:build !darwin

package tray

import (
	"context"
	"sync"

	"github.com/getlantern/systray"
	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

type systraySession struct {
	baseURL     string
	pollSession *Session
	menu        *MenuBuilder
	pollCancel  context.CancelFunc
	tooltipMu   sync.Mutex
}

func (s *systraySession) onReady() {
	systray.SetIcon(menuBarIcon())
	systray.SetTooltip("VibeGuard — no pending")

	client := api.NewClientWithBaseURL(s.baseURL)
	s.pollSession = NewSession(client)
	s.menu = NewMenuBuilder(s.pollSession)
	s.menu.Init()

	s.pollSession.OnUpdate = s.onSessionUpdate

	ctx, cancel := context.WithCancel(context.Background())
	s.pollCancel = cancel
	s.pollSession.Start(ctx)
}

func (s *systraySession) onSessionUpdate(items []api.PendingApproval, mode approvalmode.Mode, err error) {
	healthOK := err == nil
	pending := pendingCountForTitle(items, err)

	s.menu.Rebuild(items, mode, healthOK, err)
	s.setTooltip(tooltipForUpdate(items, mode, err))
	s.setTitle(0)
	s.setIcon(pending, healthOK)
}

func (s *systraySession) setIcon(pending int, healthOK bool) {
	// getlantern/systray documents SetIcon as safe from any goroutine (same as menu rebuild).
	systray.SetIcon(menuBarIconForState(pending, healthOK))
}

func (s *systraySession) setTitle(_ int) {
	// Count is rendered inside the menu-bar icon; keep title empty.
	systray.SetTitle("")
}

func (s *systraySession) setTooltip(text string) {
	s.tooltipMu.Lock()
	defer s.tooltipMu.Unlock()
	systray.SetTooltip(text)
}

func (s *systraySession) onExit() {
	if s.pollCancel != nil {
		s.pollCancel()
	}
	if s.pollSession != nil {
		s.pollSession.Stop()
	}
}
