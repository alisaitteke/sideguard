package tray

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalfmt"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
)

// PollInterval matches the TUI auto-refresh interval (internal/tui/model.go).
const PollInterval = 2 * time.Second

const apiCallTimeout = 5 * time.Second

// HistoryPageSize is the number of history rows fetched per page (initial poll and load-more).
const HistoryPageSize = 30

// TrayUpdateCallback receives poll snapshots for menu rebuild and tooltip updates.
type TrayUpdateCallback func(pending []api.PendingApproval, history []api.CommandEvent, mode approvalmode.Mode, historyHasMore bool, err error)

// Session polls the daemon HTTP API and exposes pending approvals for tray UI backends.
// Shared by systray (!darwin) and macOS NSPopover (darwin). See
// docs/plans/2026-07-01-1537-mac-tray-popover/ (mtp-phase-1.0-session.md) and
// docs/plans/2026-07-02-1226-tray-ui-polish/ (tup-phase-2.0-tray-core.md).
type Session struct {
	client *api.Client

	mu             sync.Mutex
	items          []api.PendingApproval
	history        []api.CommandEvent
	historyHasMore bool
	mode           approvalmode.Mode
	daemonDown     bool
	lastErr        error

	OnUpdate TrayUpdateCallback

	cancel    context.CancelFunc
	refreshCh chan struct{}
	wg        sync.WaitGroup
}

// NewSession creates a tray session backed by the given daemon API client.
func NewSession(client *api.Client) *Session {
	return &Session{
		client:    client,
		items:     make([]api.PendingApproval, 0),
		history:   make([]api.CommandEvent, 0),
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

// History returns a copy of the last loaded history rows (newest first).
func (s *Session) History() []api.CommandEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]api.CommandEvent, len(s.history))
	copy(out, s.history)
	return out
}

// HistoryHasMore reports whether older history rows are available via LoadMoreHistory.
func (s *Session) HistoryHasMore() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.historyHasMore
}

// LoadMoreHistory fetches the next page of older history using keyset pagination.
// Serialized with the poll loop via mu. See tup-phase-2.0-tray-core.md.
func (s *Session) LoadMoreHistory(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.daemonDown {
		return fmt.Errorf("daemon unreachable")
	}
	if len(s.history) == 0 {
		return nil
	}

	before := s.history[len(s.history)-1].CreatedAt
	callCtx, cancel := context.WithTimeout(ctx, apiCallTimeout)
	defer cancel()

	page, err := s.client.QueryEvents(callCtx, api.EventQueryParams{
		Limit:  HistoryPageSize,
		Before: before,
	})
	if err != nil {
		return err
	}

	s.history = append(s.history, page...)
	s.historyHasMore = len(page) == HistoryPageSize
	return nil
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
		items            []api.PendingApproval
		listErr          error
		mode             approvalmode.Mode = approvalmode.Ask
		modeErr          error
		history          []api.CommandEvent
		historyErr       error
		historyHasMore   bool
	)

	if healthErr == nil {
		items, listErr = s.client.ListPending(callCtx)
		if listErr == nil {
			mode, modeErr = s.client.GetApprovalMode(callCtx)
			if modeErr == nil {
				history, historyErr = s.client.QueryEvents(callCtx, api.EventQueryParams{
					Limit: HistoryPageSize,
				})
				if historyErr == nil {
					historyHasMore = len(history) == HistoryPageSize
				}
			}
		}
	}

	var (
		callback         TrayUpdateCallback
		outItems         []api.PendingApproval
		outHistory       []api.CommandEvent
		outMode          approvalmode.Mode
		outHistoryHasMore bool
		outErr           error
	)

	s.mu.Lock()
	switch {
	case healthErr != nil:
		s.daemonDown = true
		s.lastErr = healthErr
		s.items = nil
		s.history = nil
		s.historyHasMore = false
		outErr = healthErr
	case listErr != nil:
		s.daemonDown = true
		s.lastErr = listErr
		s.items = nil
		s.history = nil
		s.historyHasMore = false
		outErr = listErr
	case modeErr != nil:
		s.daemonDown = true
		s.lastErr = modeErr
		s.items = nil
		s.history = nil
		s.historyHasMore = false
		outErr = modeErr
	case historyErr != nil:
		s.daemonDown = false
		s.lastErr = historyErr
		s.items = items
		s.mode = mode
		outItems = items
		outHistory = append([]api.CommandEvent(nil), s.history...)
		outMode = mode
		outHistoryHasMore = s.historyHasMore
		outErr = historyErr
	default:
		s.daemonDown = false
		s.lastErr = nil
		s.items = items
		s.mode = mode
		s.history, s.historyHasMore = mergePolledHistory(history, historyHasMore, s.history, s.historyHasMore)
		outItems = items
		outHistory = s.history
		outMode = mode
		outHistoryHasMore = s.historyHasMore
	}
	callback = s.OnUpdate
	s.mu.Unlock()

	if callback != nil {
		callback(outItems, outHistory, outMode, outHistoryHasMore, outErr)
	}
}

// mergePolledHistory keeps older rows loaded via LoadMoreHistory when a poll tick
// refreshes the newest page. Without this, periodic polls would shrink the tray list
// and scroll restoration could re-fire proximity-based load-more paths.
func mergePolledHistory(fresh []api.CommandEvent, freshHasMore bool, prior []api.CommandEvent, priorHasMore bool) ([]api.CommandEvent, bool) {
	if len(prior) <= HistoryPageSize {
		return fresh, freshHasMore
	}
	tail := prior[HistoryPageSize:]
	freshIDs := make(map[string]struct{}, len(fresh))
	for _, ev := range fresh {
		freshIDs[ev.ID] = struct{}{}
	}
	out := append([]api.CommandEvent(nil), fresh...)
	for _, ev := range tail {
		if _, dup := freshIDs[ev.ID]; dup {
			continue
		}
		out = append(out, ev)
	}
	return out, freshHasMore || priorHasMore
}
