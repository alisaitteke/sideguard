// Runtime helpers lazily construct the LLM classifier from config.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-4.0-integration.md).
package llm

import (
	"log"
	"sync"

	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

var (
	classifierMu      sync.Mutex
	lazyClassifier    policy.Classifier
	lazyClassifierErr error
	lazyInitialized   bool
)

// ResetForTest clears the lazy classifier cache (tests only).
func ResetForTest() {
	classifierMu.Lock()
	defer classifierMu.Unlock()
	lazyClassifier = nil
	lazyClassifierErr = nil
	lazyInitialized = false
}

// Enabled reports whether LLM triage is on for cwd (global config + workspace override).
func Enabled(cwd string) bool {
	cfg, err := config.Load(cwd)
	if err != nil {
		return false
	}
	return cfg.Enabled
}

// ClassifierFor returns a Classifier when LLM is enabled for cwd, or (nil, nil) when disabled.
// On init failure, returns (nil, error) — callers should log and treat as ask (fail-safe).
func ClassifierFor(cwd string) (policy.Classifier, error) {
	cfg, err := config.Load(cwd)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, nil
	}

	classifierMu.Lock()
	defer classifierMu.Unlock()

	if !lazyInitialized {
		creds, credErr := config.ResolveCredentials()
		if credErr != nil {
			lazyClassifierErr = credErr
		} else {
			lazyClassifier, lazyClassifierErr = NewClassifier(cfg, creds)
			if lazyClassifierErr != nil {
				log.Printf("vibeguard llm: classifier init failed: %v", lazyClassifierErr)
			}
		}
		lazyInitialized = true
	}

	if lazyClassifierErr != nil {
		return nil, lazyClassifierErr
	}
	return lazyClassifier, nil
}
