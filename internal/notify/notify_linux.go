// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

//go:build linux

package notify

import (
	"fmt"
	"os/exec"
)

// sendMacOS delivers a desktop notification on Linux via notify-send.
// See docs/plans/2026-07-01-0127-sideguard-foundation/ (vgf-phase-8.0-hardening.md).
func sendMacOS(title, body string) error {
	title = truncate(title, maxTitleLen)
	body = truncate(body, maxBodyLen)

	path, err := exec.LookPath("notify-send")
	if err != nil {
		return fmt.Errorf("notify-send not found: %w", err)
	}

	cmd := exec.Command(path, title, body)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("notify-send: %w", err)
	}
	return nil
}
