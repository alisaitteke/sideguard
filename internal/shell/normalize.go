// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package shell

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// zeroWidth are invisible/format runes that attackers insert to break naive
// keyword matching (e.g. "c\u200burl"). They carry no shell semantics, so we
// strip them before parsing.
var zeroWidth = map[rune]struct{}{
	'\u200B': {}, // zero width space
	'\u200C': {}, // zero width non-joiner
	'\u200D': {}, // zero width joiner
	'\u2060': {}, // word joiner
	'\uFEFF': {}, // zero width no-break space / BOM
	'\u00AD': {}, // soft hyphen
	'\u180E': {}, // mongolian vowel separator
}

// Normalize applies Unicode NFKC normalization and strips zero-width/format
// runes so that visually-identical or invisible-character obfuscations collapse
// to a canonical form before parsing. It never changes command semantics.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-1.0-shell.md).
func Normalize(command string) string {
	if command == "" {
		return command
	}
	normalized := norm.NFKC.String(command)
	if !strings.ContainsFunc(normalized, isZeroWidth) {
		return normalized
	}
	var b strings.Builder
	b.Grow(len(normalized))
	for _, r := range normalized {
		if isZeroWidth(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isZeroWidth(r rune) bool {
	_, ok := zeroWidth[r]
	return ok
}
