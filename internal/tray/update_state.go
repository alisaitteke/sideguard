package tray

import (
	"fmt"
	"sync"

	"github.com/alisaitteke/vibeguard/internal/update"
)

// UpdateUIState is the tray-visible self-update snapshot shared by popover and systray.
// See docs/plans/2026-07-02-1102-github-update/ (vgu-phase-4.0-tray-update.md).
type UpdateUIState struct {
	Available  bool
	Version    string
	Label      string
	Installing bool
}

// UpdateState holds thread-safe update UI state for panel and menu builders.
type UpdateState struct {
	mu              sync.RWMutex
	state           UpdateUIState
	dismissedVersion string
}

// NewUpdateState creates an empty update UI state store.
func NewUpdateState() *UpdateState {
	return &UpdateState{}
}

// Get returns a copy of the current UI state.
func (s *UpdateState) Get() UpdateUIState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Set replaces the UI state.
func (s *UpdateState) Set(ui UpdateUIState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = ui
}

// TryBeginInstall marks installing when not already installing. Returns false on double-click.
func (s *UpdateState) TryBeginInstall() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Installing || !s.state.Available {
		return false
	}
	s.state.Installing = true
	return true
}

// SetInstalling updates the installing flag without changing other fields.
func (s *UpdateState) SetInstalling(installing bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Installing = installing
}

// Dismiss hides the given version until a newer release is detected.
func (s *UpdateState) Dismiss(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dismissedVersion = version
	if s.state.Version == version {
		s.state = UpdateUIState{}
	}
}

func (s *UpdateState) isDismissed(version string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dismissedVersion != "" && s.dismissedVersion == version
}

// ApplyCheckResult maps a GitHub check result into UI state and returns the new snapshot.
func (s *UpdateState) ApplyCheckResult(result update.CheckResult) UpdateUIState {
	ui := UpdateUIState{}
	if result.UpdateAvailable && result.Latest != "" && !s.isDismissed(result.Latest) {
		ui.Available = true
		ui.Version = result.Latest
		ui.Label = fmt.Sprintf("Update available: v%s", result.Latest)
	}
	s.Set(ui)
	return ui
}
