//go:build !darwin

package tray

import (
	"context"
	"sync"

	"github.com/getlantern/systray"
	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalfmt"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

type systraySession struct {
	baseURL       string
	version       string
	pollSession   *Session
	menu          *MenuBuilder
	updateChecker *UpdateChecker
	updateState   *UpdateState
	pollCancel    context.CancelFunc
	tooltipMu     sync.Mutex
}

func (s *systraySession) onReady() {
	systray.SetIcon(menuBarIcon())
	systray.SetTooltip("VibeGuard — no pending")

	client := api.NewClientWithBaseURL(s.baseURL)
	s.pollSession = NewSession(client)
	s.updateState = NewUpdateState()

	quit := func() {
		s.stop()
		systray.Quit()
	}

	s.menu = NewMenuBuilder(s.pollSession, s.updateState, func() {
		s.menu.SetUpdateUI(s.updateState.Get())
		HandleInstallUpdate(s.updateState, quit)
	}, quit)
	s.menu.Init()

	s.pollSession.OnUpdate = s.onSessionUpdate

	ctx, cancel := context.WithCancel(context.Background())
	s.pollCancel = cancel

	checker, err := NewUpdateChecker(s.version, s.updateState, func(ui UpdateUIState) {
		s.menu.SetUpdateUI(ui)
	})
	if err == nil {
		s.updateChecker = checker
		s.updateChecker.Start(ctx)
	}

	s.pollSession.Start(ctx)
}

func (s *systraySession) onSessionUpdate(items []api.PendingApproval, history []api.CommandEvent, mode approvalmode.Mode, historyHasMore bool, err error) {
	healthOK := err == nil
	pending := pendingCountForTitle(items, err)

	snapshot := PanelSnapshot{
		Items:          items,
		History:        history,
		HistoryHasMore: historyHasMore,
		Mode:           mode,
		HealthOK:       healthOK,
		Err:            err,
		Home:           approvalfmt.HomeDir(),
		Update:         s.updateState.Get(),
	}
	content := BuildTrayContent(snapshot)

	s.menu.Rebuild(content, mode, healthOK, err)
	s.setTooltip(tooltipForUpdate(items, mode, err))
	s.setTitle(0)
	s.setIcon(pending, healthOK)
}

func (s *systraySession) setIcon(pending int, healthOK bool) {
	systray.SetIcon(menuBarIconForState(pending, healthOK))
}

func (s *systraySession) setTitle(_ int) {
	systray.SetTitle("")
}

func (s *systraySession) setTooltip(text string) {
	s.tooltipMu.Lock()
	defer s.tooltipMu.Unlock()
	systray.SetTooltip(text)
}

func (s *systraySession) stop() {
	if s.pollCancel != nil {
		s.pollCancel()
	}
	if s.updateChecker != nil {
		s.updateChecker.Stop()
	}
	if s.pollSession != nil {
		s.pollSession.Stop()
	}
}

func (s *systraySession) onExit() {
	s.stop()
}
