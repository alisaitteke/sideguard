// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Package bootstrap writes default SideGuard config files on first install.
package bootstrap

import (
	"github.com/alisaitteke/sideguard/internal/config"
	"github.com/alisaitteke/sideguard/internal/detect/rules"
	"github.com/alisaitteke/sideguard/internal/llm"
	"github.com/alisaitteke/sideguard/internal/paths"
)

// EnsureDefaults writes config.yaml, signatures/default.yaml, and embedded detect
// rules when missing. Idempotent: existing files are left unchanged.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-5.0-history-cli.md).
func EnsureDefaults() error {
	if _, err := config.EnsureDefault(); err != nil {
		return err
	}
	if _, err := llm.EnsureDefaultSignature(); err != nil {
		return err
	}
	return EnsureDetectRules()
}

// EnsureDetectRules writes embedded detect YAML packs to ~/.sideguard/rules/ when
// each file is absent. User-edited files are never overwritten.
func EnsureDetectRules() error {
	dir, err := paths.RulesDir()
	if err != nil {
		return err
	}
	return rules.WriteDefaults(dir)
}
