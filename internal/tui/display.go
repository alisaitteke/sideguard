// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package tui

// ClampSelection keeps cursor in range for the given item count.
func ClampSelection(cursor, count int) int {
	if count == 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= count {
		return count - 1
	}
	return cursor
}
