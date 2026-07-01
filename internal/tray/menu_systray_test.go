//go:build !darwin

package tray

import "testing"

func TestSetModeCheckboxState(t *testing.T) {
	t.Parallel()
	// setModeCheckbox is a no-op with nil item; documents expected API usage.
	setModeCheckbox(nil, true)
	setModeCheckbox(nil, false)
}
