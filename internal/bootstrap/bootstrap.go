// Package bootstrap writes default VibeGuard config files on first install.
package bootstrap

import (
	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/llm"
)

// EnsureDefaults writes config.yaml and signatures/default.yaml when missing.
// Idempotent: existing files are left unchanged.
func EnsureDefaults() error {
	if _, err := config.EnsureDefault(); err != nil {
		return err
	}
	_, err := llm.EnsureDefaultSignature()
	return err
}
