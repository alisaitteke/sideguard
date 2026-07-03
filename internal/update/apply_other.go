// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

//go:build !darwin && !linux && !windows

package update

func newPlatformApplier() PlatformApplier {
	return NoopPlatformApplier{}
}
