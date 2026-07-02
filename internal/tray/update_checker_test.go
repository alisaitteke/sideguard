package tray

import (
	"context"
	"testing"
	"time"

	"github.com/alisaitteke/vibeguard/internal/update"
)

func TestUpdateChecker_SkipWhenDev(t *testing.T) {
	t.Parallel()

	checker, err := update.NewChecker("dev", update.Options{})
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	u := &UpdateChecker{checker: checker}
	if u.ShouldRun() {
		t.Fatal("expected dev build to skip auto-check")
	}
}

func TestUpdateChecker_SkipWhenDisabled(t *testing.T) {
	t.Parallel()

	checker, err := update.NewChecker("1.0.0", update.Options{Disabled: true})
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	u := &UpdateChecker{checker: checker}
	if u.ShouldRun() {
		t.Fatal("expected disabled checker to skip auto-check")
	}
}

func TestUpdateChecker_CallbackOnUpdate(t *testing.T) {
	t.Parallel()

	checker, err := update.NewChecker("1.0.0", update.Options{})
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	done := make(chan UpdateUIState, 1)
	u := &UpdateChecker{
		checker:      checker,
		state:        NewUpdateState(),
		interval:     time.Hour,
		initialDelay: 0,
		sleepFn:      func(time.Duration) {},
		onChange: func(ui UpdateUIState) {
			select {
			case done <- ui:
			default:
			}
		},
		checkFn: func(context.Context) (update.CheckResult, error) {
			return update.CheckResult{
				Current:         "1.0.0",
				Latest:          "2.0.0",
				UpdateAvailable: true,
			}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	u.Start(ctx)

	select {
	case ui := <-done:
		if !ui.Available {
			t.Fatalf("available = false, want true")
		}
		if ui.Version != "2.0.0" {
			t.Fatalf("version = %q, want 2.0.0", ui.Version)
		}
		if ui.Label != "Update available: v2.0.0" {
			t.Fatalf("label = %q", ui.Label)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for update callback")
	}

	u.Stop()
}

func TestUpdateState_TryBeginInstall(t *testing.T) {
	t.Parallel()

	state := NewUpdateState()
	state.Set(UpdateUIState{Available: true, Version: "2.0.0"})

	if !state.TryBeginInstall() {
		t.Fatal("first install attempt should succeed")
	}
	if state.TryBeginInstall() {
		t.Fatal("second install attempt should be blocked")
	}
	if !state.Get().Installing {
		t.Fatal("expected installing=true")
	}
}

func TestUpdateState_DismissHidesVersion(t *testing.T) {
	t.Parallel()

	state := NewUpdateState()
	result := update.CheckResult{
		Current:         "1.0.0",
		Latest:          "2.0.0",
		UpdateAvailable: true,
	}
	ui := state.ApplyCheckResult(result)
	if !ui.Available {
		t.Fatal("expected update visible before dismiss")
	}

	state.Dismiss("2.0.0")
	ui = state.ApplyCheckResult(result)
	if ui.Available {
		t.Fatal("expected dismissed version to stay hidden")
	}
}

func TestParseUpdateInterval(t *testing.T) {
	t.Parallel()

	if got := parseUpdateInterval("12h"); got != 12*time.Hour {
		t.Fatalf("12h = %v", got)
	}
	if got := parseUpdateInterval("invalid"); got != defaultUpdateCheckInterval {
		t.Fatalf("invalid = %v, want default", got)
	}
}
