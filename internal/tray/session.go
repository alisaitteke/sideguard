package tray

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalfmt"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

// PollInterval matches the TUI auto-refresh interval (internal/tui/model.go).
const PollInterval = 2 * time.Second

const apiCallTimeout = 5 * time.Second

// UpdateCallback receives poll snapshots for menu rebuild and tooltip updates.
type UpdateCallback func(items []api.PendingApproval, mode approvalmode.Mode, err error)

// Session polls the daemon HTTP API and exposes pending approvals for tray UI backends.
// Shared by systray (!darwin) and macOS NSPopover (darwin). See
// docs/plans/2026-07-01-1537-mac-tray-popover/ (mtp-phase-1.0-session.md).
type Session struct {
	client *api.Client

	mu         sync.Mutex
	items      []api.PendingApproval
	mode       approvalmode.Mode
	daemonDown bool
	lastErr    error

	OnUpdate UpdateCallback

	cancel    context.CancelFunc
	refreshCh chan struct{}
	wg        sync.WaitGroup
}

// NewSession creates a tray session backed by the given daemon API client.
func NewSession(client *api.Client) *Session {
	return &Session{
		client:    client,
		items:     make([]api.PendingApproval, 0),
		mode:      approvalmode.Ask,
		refreshCh: make(chan struct{}, 1),
	}
}

// DetectNewPending reports whether next has more pending items than prev or contains
// an approval ID not present in prev. Used for macOS popover auto-open (Phase 4).
// Returns false when both slices are empty/nil, when counts decrease, or when the
// ID set is unchanged (order-independent).
func DetectNewPending(prev, next []api.PendingApproval) bool {
	if len(next) == 0 {
		return false
	}
	if len(next) > len(prev) {
		return true
	}

	prevIDs := make(map[string]struct{}, len(prev))
	for _, p := range prev {
		prevIDs[p.ID] = struct{}{}
	}
	for _, n := range next {
		if _, ok := prevIDs[n.ID]; !ok {
			return true
		}
	}
	return false
}

// RefreshNow requests an immediate poll tick (coalesced if one is already queued).
func (s *Session) RefreshNow() {
	select {
	case s.refreshCh <- struct{}{}:
	default:
	}
}

// Start begins the background poll loop until Stop is called or ctx is cancelled.
func (s *Session) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)
	s.wg.Add(1)
	go s.loop(ctx)
}

// Stop cancels the poll loop and waits for the goroutine to exit.
func (s *Session) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// Pending returns a copy of the last successful pending list.
func (s *Session) Pending() []api.PendingApproval {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]api.PendingApproval, len(s.items))
	copy(out, s.items)
	return out
}

// Mode returns the last polled approval mode.
func (s *Session) Mode() approvalmode.Mode {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

// Healthy reports whether the last poll reached a healthy daemon.
func (s *Session) Healthy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.daemonDown
}

// Decide posts allow/deny for an approval id. Serialized with the poll loop via mu.
func (s *Session) Decide(ctx context.Context, id, decision string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, apiCallTimeout)
	defer cancel()

	_, err := s.client.Decide(callCtx, id, decision, "")
	if err != nil {
		if api.IsNotFound(err) {
			return fmt.Errorf("approval %s no longer pending", approvalfmt.ShortApprovalID(id))
		}
		return err
	}
	return nil
}

// SetMode updates the daemon global approval mode.
func (s *Session) SetMode(ctx context.Context, mode approvalmode.Mode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, apiCallTimeout)
	defer cancel()

	if err := s.client.SetApprovalMode(callCtx, mode); err != nil {
		return err
	}
	s.mode = mode
	return nil
}

func (s *Session) loop(ctx context.Context) {
	defer s.wg.Done()

	s.tick(ctx)

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.refreshCh:
			s.tick(ctx)
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Session) tick(ctx context.Context) {
	callCtx, cancel := context.WithTimeout(ctx, apiCallTimeout)
	defer cancel()

	_, healthErr := s.client.Health(callCtx)

	var (
		items   []api.PendingApproval
		listErr error
		mode    approvalmode.Mode = approvalmode.Ask
		modeErr error
	)
	if healthErr == nil {
		items, listErr = s.client.ListPending(callCtx)
		if listErr == nil {
			mode, modeErr = s.client.GetApprovalMode(callCtx)
		}
	}

	var (
		callback UpdateCallback
		outItems []api.PendingApproval
		outMode  approvalmode.Mode
		outErr   error
	)

	s.mu.Lock()
	switch {
	case healthErr != nil:
		s.daemonDown = true
		s.lastErr = healthErr
		s.items = nil
		outErr = healthErr
	case listErr != nil:
		s.daemonDown = true
		s.lastErr = listErr
		s.items = nil
		outErr = listErr
	case modeErr != nil:
		s.daemonDown = true
		s.lastErr = modeErr
		s.items = nil
		outErr = modeErr
	default:
		s.daemonDown = false
		s.lastErr = nil
		s.items = items
		s.mode = mode
		outItems = items
		outMode = mode
	}
	callback = s.OnUpdate
	s.mu.Unlock()

	if callback != nil {
		callback(outItems, outMode, outErr)
	}
}
