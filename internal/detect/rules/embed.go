// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Package rules embeds SideGuard's built-in detect rule packs so the daemon
// ships with a working ruleset regardless of what is present on disk. The YAML
// files are the trusted source for bypass (self-protection) rules, which user
// rule packs may never override.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-2.0-detect.md).
package rules

import "embed"

// FS holds the embedded rule pack YAML files (one file per category group).
//
//go:embed *.yaml
var FS embed.FS
