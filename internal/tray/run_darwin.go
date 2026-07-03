// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

//go:build darwin

package tray

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalfmt"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
	"github.com/alisaitteke/sideguard/internal/llm"
	"github.com/alisaitteke/sideguard/internal/tray/darwin"
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
	var loadMoreLoading sync.Mutex
	var lastRenderedPanel []byte

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

		content := BuildTrayContent(snapshot)

		payload := darwin.PanelJSON{
			ModeIndex:       content.ModeIndex,
			ModeEnabled:     content.ModeEnabled,
			PendingRows:     trayRowsToPanelJSON(content.PendingRows),
			HistoryRows:     trayRowsToPanelJSON(content.HistoryRows),
			HistoryHasMore:  content.HistoryHasMore,
			PendingOverflow: content.PendingOverflow,
			EmptyMessage:    content.EmptyMessage,
			FooterDaemon:    content.FooterDaemon,
			FooterPending:   content.FooterPending,
			UpdateVisible:   content.UpdateVisible,
			UpdateLabel:     content.UpdateLabel,
			UpdateEnabled:   content.UpdateEnabled,
		}

		encoded, err := json.Marshal(payload)
		if err != nil {
			return
		}
		if bytes.Equal(encoded, lastRenderedPanel) {
			return
		}
		lastRenderedPanel = encoded

		darwin.UpdatePanel(payload)
	}

	var prevPending []api.PendingApproval

	pollSession.OnUpdate = func(items []api.PendingApproval, history []api.CommandEvent, mode approvalmode.Mode, historyHasMore bool, err error) {
		pending := pendingCountForTitle(items, err)
		healthOK := err == nil

		if healthOK && DetectNewPending(prevPending, items) {
			darwin.ShowPopover()
		}

		panelMu.Lock()
		lastSnapshot = PanelSnapshot{
			Items:          items,
			History:        history,
			HistoryHasMore: historyHasMore,
			Mode:           mode,
			HealthOK:       healthOK,
			Err:            err,
			Home:           home,
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
			darwin.SetTooltip("SideGuard — mode change failed: " + err.Error())
			return
		}
		pollSession.RefreshNow()
	}

	darwin.SetDecideHandler(onDecide)
	darwin.SetModeHandler(onSetMode)
	darwin.SetLoadMoreHandler(func() {
		if !loadMoreLoading.TryLock() {
			return
		}
		defer loadMoreLoading.Unlock()

		loadCtx, loadCancel := context.WithTimeout(ctx, apiCallTimeout)
		defer loadCancel()
		if err := pollSession.LoadMoreHistory(loadCtx); err != nil {
			return
		}
		panelMu.Lock()
		lastSnapshot.History = pollSession.History()
		lastSnapshot.HistoryHasMore = pollSession.HistoryHasMore()
		panelMu.Unlock()
		refreshPanel()
	})
	darwin.SetInstallHandler(func() {
		refreshPanel()
		HandleInstallUpdate(updateUIState, quitTray)
	})
	darwin.SetQuitHandler(quitTray)

	darwin.SetOpenSettingsHandler(func() {
		snap, err := darwin.LoadSettingsSnapshot()
		if err != nil {
			darwin.SetTooltip("SideGuard — settings load failed: " + err.Error())
			return
		}
		darwin.ShowSettings(snap)
	})
	darwin.SetSaveSettingsHandler(func(payload string) error {
		if err := darwin.SaveSettingsFromJSON(payload); err != nil {
			return err
		}
		llm.ResetClassifierCache()
		return nil
	})
	darwin.SetAnalyseHandler(func(rowID, command string, useEventID bool) darwin.AnalyseResultJSON {
		analyseCtx, analyseCancel := context.WithTimeout(ctx, 30*time.Second)
		defer analyseCancel()
		return darwin.RunAnalyze(analyseCtx, client, rowID, command, useEventID)
	})

	darwin.SetAppearanceHandler(func(dark bool) {
		darwin.SetHeaderLogo(popoverHeaderLogo(dark))
	})

	readyCh := make(chan struct{})
	darwin.SetReadyHandler(func() {
		close(readyCh)
	})
	go func() {
		<-readyCh
		// Initial logo is set from ObjC after effective appearance is resolved.
		darwin.SetIcon(menuBarIcon())
		darwin.SetTooltip("SideGuard — no pending")
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

func trayRowsToPanelJSON(rows []TrayRow) []darwin.PanelJSONRow {
	out := make([]darwin.PanelJSONRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, darwin.PanelJSONRow{
			Kind:   string(row.Kind),
			ID:     row.ID,
			Label:  row.Label,
			Detail: row.Detail,
		})
	}
	return out
}
