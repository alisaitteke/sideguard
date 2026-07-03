// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Auth mode validation for LLM provider instances.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-2.0-llm.md).
package llm

import (
	"errors"
	"fmt"
)

// ErrAuthNotImplemented is returned when auth_mode subscription is requested.
var ErrAuthNotImplemented = errors.New("subscription auth_mode is not implemented yet")

// ValidateAuthMode checks that the provider instance auth mode is supported at runtime.
func ValidateAuthMode(mode string) error {
	switch mode {
	case "api_key", "":
		return nil
	case "subscription":
		return ErrAuthNotImplemented
	default:
		return fmt.Errorf("unknown auth_mode %q", mode)
	}
}
