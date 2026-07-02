//go:build !darwin

package tray

import (
	"context"
	"fmt"
	"sync"

	"github.com/getlantern/systray"
	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalfmt"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

// MenuBuilder owns the systray context menu and rebuilds approval rows on each poll.
// See docs/plans/2026-07-01-1515-global-approval-mode/ (gam-phase-3.0-tray-menu.md).
type MenuBuilder struct {
	session     *Session
	home        string
	updateState *UpdateState
	onInstall   func()
	onQuit      func()

	daemonStatus  *systray.MenuItem
	pendingStatus *systray.MenuItem
	modeStatus    *systray.MenuItem
	modeAsk       *systray.MenuItem
	modeAuto      *systray.MenuItem
	modeAutoSub   *systray.MenuItem
	modeAutoAllow *systray.MenuItem
	modeAutoDeny  *systray.MenuItem
	overflow      *systray.MenuItem
	refresh       *systray.MenuItem
	updateItem    *systray.MenuItem
	slots         []approvalSlot

	deciding sync.Map // approval id while Decide is in flight
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
		home:        approvalfmt.HomeDir(),
		updateState: updateState,
		onInstall:   onInstall,
		onQuit:      onQuit,
	}
}

// Init builds the static menu shell and approval slots. Call once from onReady.
func (mb *MenuBuilder) Init() {
	header := systray.AddMenuItem("VibeGuard", "VibeGuard menu-bar tray")
	header.Disable()

	systray.AddSeparator()

	mb.daemonStatus = systray.AddMenuItem(formatDaemonStatus(true, nil), "Daemon connection status")
	mb.daemonStatus.Disable()

	mb.pendingStatus = systray.AddMenuItem(formatPendingCount(0, true), "Pending approval count")
	mb.pendingStatus.Disable()

	mb.modeStatus = systray.AddMenuItem(formatModeStatus(approvalmode.Ask, true), "Global approval mode")
	mb.modeStatus.Disable()

	systray.AddSeparator()

	modeMenu := systray.AddMenuItem("Mode", "Global approval mode")
	mb.modeAsk = modeMenu.AddSubMenuItemCheckbox("Ask", "Manual Allow/Deny for each request", true)
	mb.modeAuto = modeMenu.AddSubMenuItemCheckbox("Auto", "Smart triage: safe commands pass, risky blocked, uncertain queue", false)
	mb.modeAutoSub = modeMenu.AddSubMenuItem("Auto-decide", "Blanket auto approval decisions")
	mb.modeAutoAllow = mb.modeAutoSub.AddSubMenuItemCheckbox("Approve", "Auto-allow all queued requests", false)
	mb.modeAutoDeny = mb.modeAutoSub.AddSubMenuItemCheckbox("Deny", "Auto-deny all queued requests", false)
	mb.wireModeItems()

	systray.AddSeparator()

	mb.slots = make([]approvalSlot, maxVisiblePending)
	for i := range mb.slots {
		slot := &mb.slots[i]
		label := fmt.Sprintf("Pending %d", i+1)
		slot.parent = systray.AddMenuItem(label, "Pending approval")
		slot.allow = slot.parent.AddSubMenuItem("Allow", "Allow this command")
		slot.deny = slot.parent.AddSubMenuItem("Deny", "Deny this command")
		slot.parent.Hide()
		mb.wireSlot(i)
	}

	mb.overflow = systray.AddMenuItem("", "More pending approvals")
	mb.overflow.Disable()
	mb.overflow.Hide()

	systray.AddSeparator()

	mb.refresh = systray.AddMenuItem("Refresh", "Poll pending approvals now")
	go func() {
		for range mb.refresh.ClickedCh {
			mb.session.RefreshNow()
		}
	}()

	terminalUI := systray.AddMenuItem("Open Terminal UI…", "Run: vibeguard ui")
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

	quit := systray.AddMenuItem("Quit", "Exit the menu-bar tray")
	go func() {
		<-quit.ClickedCh
		if mb.onQuit != nil {
			mb.onQuit()
			return
		}
		systray.Quit()
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

// Rebuild updates status rows and visible approval slots from the latest poll snapshot.
func (mb *MenuBuilder) Rebuild(items []api.PendingApproval, mode approvalmode.Mode, healthOK bool, err error) {
	mb.daemonStatus.SetTitle(formatDaemonStatus(healthOK, err))
	mb.pendingStatus.SetTitle(formatPendingCount(len(items), healthOK))
	mb.modeStatus.SetTitle(formatModeStatus(mode, healthOK))
	if healthOK {
		mb.SetModeUI(mode)
	}

	visible, overflow := visiblePendingItems(items, maxVisiblePending)

	for i := range mb.slots {
		slot := &mb.slots[i]
		if i < len(visible) {
			item := visible[i]
			slot.currentID = item.ID
			slot.parent.SetTitle(truncateMenuLabel(approvalfmt.FormatListLine(item, mb.home), maxPanelLabelLen))
			slot.parent.Show()
		} else {
			slot.currentID = ""
			slot.parent.Hide()
		}
	}

	if overflow > 0 {
		mb.overflow.SetTitle(overflowLabel(overflow))
		mb.overflow.Show()
	} else {
		mb.overflow.Hide()
	}

	mb.refreshUpdateItem()
}

// SetUpdateUI shows or hides the Install update menu item above Quit.
func (mb *MenuBuilder) SetUpdateUI(ui UpdateUIState) {
	if mb == nil || mb.updateItem == nil {
		return
	}
	if ui.Available {
		title := fmt.Sprintf("Install update v%s…", ui.Version)
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

func (mb *MenuBuilder) refreshUpdateItem() {
	if mb.updateState == nil {
		return
	}
	mb.SetUpdateUI(mb.updateState.Get())
}
