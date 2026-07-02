//go:build darwin

package tray

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalfmt"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
	"github.com/alisaitteke/vibeguard/internal/tray/darwin"
)

// Run starts the macOS menu-bar tray (NSStatusItem + NSPopover) and blocks until Quit.
// Requires CGO_ENABLED=1 and an active GUI session.
// See docs/plans/2026-07-01-1537-mac-tray-popover/ (mtp-phase-4.0-auto-open.md).
func Run(opts Options) error {
	baseURL := resolveBaseURL(opts)
	client := api.NewClientWithBaseURL(baseURL)
	pollSession := NewSession(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var deciding sync.Map

	home := approvalfmt.HomeDir()
	version := opts.Version
	if version == "" {
		version = "dev"
	}

	updateUIState := NewUpdateState()
	var panelMu sync.Mutex
	var lastSnapshot PanelSnapshot

	refreshPanel := func() {
		panelMu.Lock()
		snapshot := lastSnapshot
		snapshot.Update = updateUIState.Get()
		panelMu.Unlock()

		content := BuildPanelRows(snapshot)

		rows := make([]darwin.PanelJSONRow, 0, len(content.Rows))
		for _, row := range content.Rows {
			rows = append(rows, darwin.PanelJSONRow{
				ID:    row.ID,
				Label: row.Label,
			})
		}

		darwin.UpdatePanel(darwin.PanelJSON{
			DaemonStatus:  content.DaemonStatus,
			PendingCount:  content.PendingCount,
			ModeIndex:     content.ModeIndex,
			ModeEnabled:   content.ModeEnabled,
			Rows:          rows,
			OverflowHint:  content.OverflowHint,
			EmptyMessage:  content.EmptyMessage,
			UpdateVisible: content.UpdateVisible,
			UpdateLabel:   content.UpdateLabel,
			UpdateEnabled: content.UpdateEnabled,
		})
	}

	var prevPending []api.PendingApproval

	pollSession.OnUpdate = func(items []api.PendingApproval, mode approvalmode.Mode, err error) {
		pending := pendingCountForTitle(items, err)
		healthOK := err == nil

		if healthOK && DetectNewPending(prevPending, items) {
			darwin.ShowPopover()
		}

		panelMu.Lock()
		lastSnapshot = PanelSnapshot{
			Items:    items,
			Mode:     mode,
			HealthOK: healthOK,
			Err:      err,
			Home:     home,
		}
		panelMu.Unlock()

		refreshPanel()

		darwin.SetIcon(menuBarIconForState(pending, healthOK))
		darwin.SetTitle("")
		darwin.SetTooltip(tooltipForUpdate(items, mode, err))

		if healthOK {
			prevPending = make([]api.PendingApproval, len(items))
			copy(prevPending, items)
		}
	}

	updateChecker, err := NewUpdateChecker(version, updateUIState, func(UpdateUIState) {
		refreshPanel()
	})
	if err != nil {
		return err
	}

	quitTray := func() {
		cancel()
		updateChecker.Stop()
		pollSession.Stop()
		darwin.Quit()
	}

	onDecide := func(id, decision string) {
		if id == "" {
			return
		}
		if _, loaded := deciding.LoadOrStore(id, struct{}{}); loaded {
			return
		}
		defer deciding.Delete(id)

		callCtx, callCancel := context.WithTimeout(ctx, apiCallTimeout)
		defer callCancel()

		_ = pollSession.Decide(callCtx, id, decision)
		pollSession.RefreshNow()
	}

	onSetMode := func(modeIndex int) {
		mode := ModeFromSegmentIndex(modeIndex)
		callCtx, callCancel := context.WithTimeout(ctx, apiCallTimeout)
		defer callCancel()

		if err := pollSession.SetMode(callCtx, mode); err != nil {
			darwin.SetTooltip("VibeGuard — mode change failed: " + err.Error())
			return
		}
		pollSession.RefreshNow()
	}

	darwin.SetDecideHandler(onDecide)
	darwin.SetModeHandler(onSetMode)
	darwin.SetInstallHandler(func() {
		refreshPanel()
		HandleInstallUpdate(updateUIState, quitTray)
	})
	darwin.SetQuitHandler(quitTray)

	readyCh := make(chan struct{})
	darwin.SetReadyHandler(func() {
		close(readyCh)
	})
	go func() {
		<-readyCh
		darwin.SetIcon(menuBarIcon())
		darwin.SetTooltip("VibeGuard — no pending")
		pollSession.Start(ctx)
		updateChecker.Start(ctx)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			quitTray()
		}()
	}()

	darwinPrepare()
	runDarwinAppKitLoop()

	cancel()
	updateChecker.Stop()
	pollSession.Stop()
	return nil
}
