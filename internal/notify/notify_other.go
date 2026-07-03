// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

//go:build !darwin && !linux

package notify

import "fmt"

func sendMacOS(title, body string) error {
	return fmt.Errorf("macOS notifications are not supported on %s", "non-darwin")
}
