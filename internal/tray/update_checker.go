package tray

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/alisaitteke/sideguard/internal/config"
	"github.com/alisaitteke/sideguard/internal/update"
)

const (
	defaultUpdateCheckInterval = 6 * time.Hour
	defaultInitialUpdateDelay  = 60 * time.Second
	maxInitialUpdateJitter     = 30 * time.Second
)

// UpdateChecker polls GitHub for new releases on its own goroutine (separate from Session poll).
// See docs/plans/2026-07-02-1102-github-update/ (vgu-phase-4.0-tray-update.md).
type UpdateChecker struct {
	checker  *update.Checker
	interval time.Duration
	state    *UpdateState
	onChange func(UpdateUIState)

	checkFn      func(ctx context.Context) (update.CheckResult, error)
	initialDelay time.Duration
	sleepFn      func(time.Duration)

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewUpdateChecker builds a background update checker for the tray using config and binary version.
func NewUpdateChecker(version string, state *UpdateState, onChange func(UpdateUIState)) (*UpdateChecker, error) {
	if state == nil {
		state = NewUpdateState()
	}

	updateCfg, err := config.LoadUpdate()
	if err != nil {
		return nil, err
	}

	interval := parseUpdateInterval(updateCfg.CheckInterval)
	checker, err := update.NewChecker(version, update.Options{Disabled: !updateCfg.Enabled})
	if err != nil {
		return nil, err
	}

	u := &UpdateChecker{
		checker:      checker,
		interval:     interval,
		state:        state,
		onChange:     onChange,
		initialDelay: defaultInitialUpdateDelay + time.Duration(rand.Intn(int(maxInitialUpdateJitter/time.Second)+1))*time.Second,
		sleepFn:      time.Sleep,
	}
	u.checkFn = checker.Check
	return u, nil
}

// ShouldRun reports whether background polling should start.
func (u *UpdateChecker) ShouldRun() bool {
	if u == nil || u.checker == nil {
		return false
	}
	return u.checker.ShouldAutoCheck()
}

// Start begins the background check loop until Stop is called or ctx is cancelled.
func (u *UpdateChecker) Start(ctx context.Context) {
	if u == nil || !u.ShouldRun() {
		return
	}

	ctx, u.cancel = context.WithCancel(ctx)
	u.wg.Add(1)
	go u.loop(ctx)
}

// Stop cancels the check loop and waits for the goroutine to exit.
func (u *UpdateChecker) Stop() {
	if u == nil || u.cancel == nil {
		return
	}
	u.cancel()
	u.wg.Wait()
}

// Dismiss hides the given release version until a newer one is found.
func (u *UpdateChecker) Dismiss(version string) {
	if u == nil || u.state == nil {
		return
	}
	u.state.Dismiss(version)
	if u.onChange != nil {
		u.onChange(u.state.Get())
	}
}

func (u *UpdateChecker) loop(ctx context.Context) {
	defer u.wg.Done()

	if u.initialDelay > 0 {
		if !u.sleep(ctx, u.initialDelay) {
			return
		}
	}

	u.tick(ctx)

	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.tick(ctx)
		}
	}
}

func (u *UpdateChecker) tick(ctx context.Context) {
	if u.checkFn == nil {
		return
	}

	callCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	result, err := u.checkFn(callCtx)
	if err != nil {
		log.Printf("sideguard tray: update check failed: %v", err)
		return
	}

	ui := u.state.ApplyCheckResult(result)
	if u.onChange != nil {
		u.onChange(ui)
	}
}

func (u *UpdateChecker) sleep(ctx context.Context, d time.Duration) bool {
	if u.sleepFn != nil {
		done := make(chan struct{})
		go func() {
			u.sleepFn(d)
			close(done)
		}()
		select {
		case <-ctx.Done():
			return false
		case <-done:
			return true
		}
	}

	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func parseUpdateInterval(raw string) time.Duration {
	if raw == "" {
		return defaultUpdateCheckInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return defaultUpdateCheckInterval
	}
	return d
}
