// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package update

import (
	"strings"

	"golang.org/x/mod/semver"
)

// NormalizeVersion strips a leading "v" and surrounding whitespace from a tag.
func NormalizeVersion(tag string) string {
	return strings.TrimPrefix(strings.TrimSpace(tag), "v")
}

// Compare returns semver.Compare(latest, current): positive when latest is newer.
func Compare(current, latest string) int {
	cur := semver.Canonical("v" + NormalizeVersion(current))
	lat := semver.Canonical("v" + NormalizeVersion(latest))
	if cur == "" || lat == "" {
		return 0
	}
	return semver.Compare(lat, cur)
}

// IsNewer reports whether latest is strictly newer than current.
func IsNewer(current, latest string) bool {
	return Compare(current, latest) > 0
}
