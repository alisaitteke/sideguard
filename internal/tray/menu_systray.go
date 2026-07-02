//go:build !darwin

package tray

import (
	"context"
	"fmt"
	"sync"

	"github.com/getlantern/systray"
	"github.com/alisaitteke/sideguard/internal/approvalfmt"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
)

// MenuBuilder owns the systray context menu and rebuilds approval rows on each poll.
// See docs/plans/2026-07-01-1515-global-approval-mode/ (gam-phase-3.0-tray-menu.md) and
// docs/plans/2026-07-02-1226-tray-ui-polish/ (tup-phase-4.0-tray-systray.md).
type MenuBuilder struct {
	session     *Session
	updateState *UpdateState
	onInstall   func()
	onQuit      func()

	modeAsk       *systray.MenuItem
	modeAuto      *systray.MenuItem
	modeAutoSub   *systray.MenuItem
	modeAutoAllow *systray.MenuItem
	modeAutoDeny  *systray.MenuItem
	overflow      *systray.MenuItem
	historyHeader *systray.MenuItem
	historySlots  []*systray.MenuItem
	loadMore      *systray.MenuItem
	footerDaemon  *systray.MenuItem
	footerPending *systray.MenuItem
	refresh       *systray.MenuItem
	updateItem    *systray.MenuItem
	quitConfirm   *systray.MenuItem
	slots         []approvalSlot

	deciding        sync.Map // approval id while Decide is in flight
	loadMoreLoading sync.Mutex
}

type approvalSlot struct {
	parent    *systray.MenuItem
	allow     *systray.MenuItem
	deny      *systray.MenuItem
	currentID string
}

// NewMenuBuilder creates a menu builder bound to the given session.
func NewMenuBuilder(session *Session, updateState *UpdateState, onInstall, onQuit func()) *MenuBuilder {
	return &MenuBuilder{
		session:     session,
		updateState: updateState,
		onInstall:   onInstall,
		onQuit:      onQuit,
	}
}

// Init builds the static menu shell and approval slots. Call once from onReady.
func (mb *MenuBuilder) Init() {
	header := systray.AddMenuItem("SideGuard", "SideGuard menu-bar tray")
	header.Disable()

	systray.AddSeparator()

	modeMenu := systray.AddMenuItem("Mode", "Global approval mode")
	mb.modeAsk = modeMenu.AddSubMenuItemCheckbox("Ask", "Manual Run/Decline for each request", true)
	mb.modeAuto = modeMenu.AddSubMenuItemCheckbox("Auto", "Smart triage: safe commands pass, risky blocked, uncertain queue", false)
	mb.modeAutoSub = modeMenu.AddSubMenuItem("Auto-decide", "Blanket auto approval decisions")
	mb.modeAutoAllow = mb.modeAutoSub.AddSubMenuItemCheckbox("Run", "Auto-allow all queued requests", false)
	mb.modeAutoDeny = mb.modeAutoSub.AddSubMenuItemCheckbox("Decline", "Auto-deny all queued requests", false)
	mb.wireModeItems()

	systray.AddSeparator()

	mb.slots = make([]approvalSlot, maxVisiblePending)
	for i := range mb.slots {
		slot := &mb.slots[i]
		label := fmt.Sprintf("Pending %d", i+1)
		slot.parent = systray.AddMenuItem(label, "Pending approval")
		slot.allow = slot.parent.AddSubMenuItem("Run", "Run this command")
		slot.deny = slot.parent.AddSubMenuItem("Decline", "Decline this command")
		slot.parent.Hide()
		mb.wireSlot(i)
	}

	mb.overflow = systray.AddMenuItem("", "More pending approvals")
	mb.overflow.Disable()
	mb.overflow.Hide()

	systray.AddSeparator()

	mb.historyHeader = systray.AddMenuItem("History", "Resolved approval history")
	mb.historyHeader.Disable()
	mb.historyHeader.Hide()

	mb.historySlots = make([]*systray.MenuItem, maxVisibleHistory)
	for i := range mb.historySlots {
		item := systray.AddMenuItem("", "Resolved approval")
		item.Disable()
		item.Hide()
		mb.historySlots[i] = item
	}

	mb.loadMore = systray.AddMenuItem("Load older history…", "Fetch older resolved approvals")
	mb.loadMore.Hide()
	go func() {
		for range mb.loadMore.ClickedCh {
			if !mb.loadMoreLoading.TryLock() {
				continue
			}
			go func() {
				defer mb.loadMoreLoading.Unlock()
				ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout)
				defer cancel()
				if err := mb.session.LoadMoreHistory(ctx); err != nil {
					return
				}
				mb.rebuildFromSession()
			}()
		}
	}()

	systray.AddSeparator()

	mb.footerDaemon = systray.AddMenuItem(formatDaemonStatus(true, nil), "Daemon connection status")
	mb.footerDaemon.Disable()

	mb.footerPending = systray.AddMenuItem(formatPendingCount(0, true), "Pending approval count")
	mb.footerPending.Disable()

	systray.AddSeparator()

	mb.refresh = systray.AddMenuItem("Refresh", "Poll pending approvals now")
	go func() {
		for range mb.refresh.ClickedCh {
			mb.session.RefreshNow()
		}
	}()

	terminalUI := systray.AddMenuItem("Open Terminal UI…", "Run: sideguard ui")
	terminalUI.Disable()

	systray.AddSeparator()

	mb.updateItem = systray.AddMenuItem("Install update…", "Download and install the latest release")
	mb.updateItem.Hide()
	go func() {
		for range mb.updateItem.ClickedCh {
			if mb.onInstall != nil {
				mb.onInstall()
			}
		}
	}()

	systray.AddSeparator()

	quitParent := systray.AddMenuItem("Quit SideGuard…", "Exit the menu-bar tray")
	mb.quitConfirm = quitParent.AddSubMenuItem("Quit", "Confirm quit")
	quitParent.AddSubMenuItem("Cancel", "Keep running")
	go func() {
		for range mb.quitConfirm.ClickedCh {
			if mb.onQuit != nil {
				mb.onQuit()
				return
			}
			systray.Quit()
		}
	}()
}

func (mb *MenuBuilder) wireModeItems() {
	go func() {
		for range mb.modeAsk.ClickedCh {
			mb.selectMode(approvalmode.Ask)
		}
	}()
	go func() {
		for range mb.modeAuto.ClickedCh {
			mb.selectMode(approvalmode.Auto)
		}
	}()
	go func() {
		for range mb.modeAutoAllow.ClickedCh {
			mb.selectMode(approvalmode.AutoAllow)
		}
	}()
	go func() {
		for range mb.modeAutoDeny.ClickedCh {
			mb.selectMode(approvalmode.AutoDeny)
		}
	}()
}

func (mb *MenuBuilder) selectMode(mode approvalmode.Mode) {
	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout)
	defer cancel()

	if err := mb.session.SetMode(ctx, mode); err != nil {
		return
	}
	mb.SetModeUI(mode)
	mb.session.RefreshNow()
}

// SetModeUI updates checkbox state to reflect the current daemon mode.
func (mb *MenuBuilder) SetModeUI(mode approvalmode.Mode) {
	setModeCheckbox(mb.modeAsk, mode == approvalmode.Ask)
	setModeCheckbox(mb.modeAuto, mode == approvalmode.Auto)
	setModeCheckbox(mb.modeAutoAllow, mode == approvalmode.AutoAllow)
	setModeCheckbox(mb.modeAutoDeny, mode == approvalmode.AutoDeny)
}

func setModeCheckbox(item *systray.MenuItem, checked bool) {
	if item == nil {
		return
	}
	if checked {
		item.Check()
	} else {
		item.Uncheck()
	}
}

func (mb *MenuBuilder) wireSlot(idx int) {
	go func() {
		for range mb.slots[idx].allow.ClickedCh {
			mb.onDecide(idx, "allow")
		}
	}()
	go func() {
		for range mb.slots[idx].deny.ClickedCh {
			mb.onDecide(idx, "deny")
		}
	}()
}

func (mb *MenuBuilder) onDecide(slotIdx int, decision string) {
	id := mb.slots[slotIdx].currentID
	if id == "" {
		return
	}
	if _, loaded := mb.deciding.LoadOrStore(id, struct{}{}); loaded {
		return
	}
	defer mb.deciding.Delete(id)

	ctx, cancel := context.WithTimeout(context.Background(), apiCallTimeout)
	defer cancel()

	_ = mb.session.Decide(ctx, id, decision)
	mb.session.RefreshNow()
}

// rebuildFromSession renders the menu from the current session snapshot without a poll tick.
// Used after LoadMoreHistory so appended rows are visible (RefreshNow would reset history).
func (mb *MenuBuilder) rebuildFromSession() {
	snapshot := PanelSnapshot{
		Items:          mb.session.Pending(),
		History:        mb.session.History(),
		HistoryHasMore: mb.session.HistoryHasMore(),
		Mode:           mb.session.Mode(),
		HealthOK:       mb.session.Healthy(),
		Home:           approvalfmt.HomeDir(),
	}
	if mb.updateState != nil {
		snapshot.Update = mb.updateState.Get()
	}
	content := BuildTrayContent(snapshot)
	mb.Rebuild(content, snapshot.Mode, snapshot.HealthOK, nil)
}

// Rebuild updates menu rows from BuildTrayContent output (pending, history, footer).
func (mb *MenuBuilder) Rebuild(content TrayContent, mode approvalmode.Mode, healthOK bool, err error) {
	mb.footerDaemon.SetTitle(content.FooterDaemon)
	mb.footerPending.SetTitle(content.FooterPending)
	if healthOK {
		mb.SetModeUI(mode)
	}

	for i := range mb.slots {
		slot := &mb.slots[i]
		if i < len(content.PendingRows) {
			row := content.PendingRows[i]
			slot.currentID = row.ID
			slot.parent.SetTitle(row.Label)
			slot.parent.Show()
		} else {
			slot.currentID = ""
			slot.parent.Hide()
		}
	}

	if content.PendingOverflow != "" {
		mb.overflow.SetTitle(content.PendingOverflow)
		mb.overflow.Show()
	} else {
		mb.overflow.Hide()
	}

	visibleHistory, showHistory, showLoadMore := visibleSystrayHistory(content.HistoryRows, content.HistoryHasMore, maxVisibleHistory)
	if showHistory {
		mb.historyHeader.Show()
	} else {
		mb.historyHeader.Hide()
	}
	for i, slot := range mb.historySlots {
		if showHistory && i < len(visibleHistory) {
			slot.SetTitle(visibleHistory[i].Label)
			slot.Show()
		} else {
			slot.Hide()
		}
	}
	if showLoadMore {
		mb.loadMore.Show()
		mb.loadMore.Enable()
	} else {
		mb.loadMore.Hide()
	}

	if mb.updateState != nil {
		mb.SetUpdateUI(mb.updateState.Get())
	}
}

// SetUpdateUI shows or hides the Install update menu item above Quit.
func (mb *MenuBuilder) SetUpdateUI(ui UpdateUIState) {
	if mb == nil || mb.updateItem == nil {
		return
	}
	if ui.Available {
		title := ui.Label
		if title == "" && ui.Version != "" {
			title = fmt.Sprintf("Install update v%s…", ui.Version)
		}
		if title == "" {
			title = "Install update…"
		}
		mb.updateItem.SetTitle(title)
		mb.updateItem.Show()
		if ui.Installing {
			mb.updateItem.Disable()
		} else {
			mb.updateItem.Enable()
		}
		return
	}
	mb.updateItem.Hide()
}
