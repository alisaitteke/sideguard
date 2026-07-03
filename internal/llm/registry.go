// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Driver registry for extensible LLM backends.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-2.0-llm.md).
package llm

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/alisaitteke/sideguard/internal/config"
)

// DriverInfo describes a registered LLM driver for settings UI dropdowns.
type DriverInfo struct {
	Name            string
	Label           string
	SupportsBaseURL bool
	AuthModes       []string
}

// DriverFactory constructs a ChatDriver for a resolved provider instance.
type DriverFactory func(cfg driverConfig) (ChatDriver, error)

type driverConfig struct {
	instance config.ProviderInstance
	apiKey   string
	timeout  time.Duration
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]DriverFactory)
	driverMeta = make(map[string]DriverInfo)
)

func init() {
	registerDriver("openai", "OpenAI", true, []string{"api_key", "subscription"}, newOpenAIChatDriver)
	registerDriver("openai-compatible", "OpenAI-compatible", true, []string{"api_key", "subscription"}, newOpenAIChatDriver)
	registerDriver("anthropic", "Anthropic", true, []string{"api_key", "subscription"}, newAnthropicChatDriver)
	registerDriver("ollama", "Ollama", true, []string{"api_key"}, newOllamaChatDriver)
}

func registerDriver(name, label string, supportsBaseURL bool, authModes []string, factory DriverFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
	driverMeta[name] = DriverInfo{
		Name:            name,
		Label:           label,
		SupportsBaseURL: supportsBaseURL,
		AuthModes:       append([]string(nil), authModes...),
	}
}

// Register adds or replaces a driver factory (primarily for tests).
func Register(driver string, factory DriverFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[driver] = factory
	if _, ok := driverMeta[driver]; !ok {
		driverMeta[driver] = DriverInfo{Name: driver, Label: driver, SupportsBaseURL: true, AuthModes: []string{"api_key"}}
	}
}

// RegisteredDrivers returns stable metadata for all built-in drivers.
func RegisteredDrivers() []DriverInfo {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]DriverInfo, 0, len(driverMeta))
	for _, info := range driverMeta {
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// NewChatDriver constructs a ChatDriver for a provider instance and credential.
func NewChatDriver(instance config.ProviderInstance, cred config.ProviderCredential, timeoutMS int) (ChatDriver, error) {
	if err := ValidateAuthMode(instance.AuthMode); err != nil {
		return nil, fmt.Errorf("provider %q: %w", instance.ID, err)
	}

	if instance.Driver != "ollama" && cred.APIKey == "" {
		return nil, fmt.Errorf("provider %q: API key required", instance.ID)
	}

	registryMu.RLock()
	factory, ok := registry[instance.Driver]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown driver %q for provider %q", instance.Driver, instance.ID)
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	return factory(driverConfig{
		instance: instance,
		apiKey:   cred.APIKey,
		timeout:  timeout,
	})
}

// resolveProviderInstance finds a provider by id or returns the default.
func resolveProviderInstance(settings config.LLMSettings, providerID string) (config.ProviderInstance, error) {
	if providerID == "" {
		providerID = settings.DefaultProvider
	}
	if providerID == "" {
		return config.ProviderInstance{}, fmt.Errorf("no default_provider configured")
	}

	for _, p := range settings.Providers {
		if p.ID == providerID {
			return p, nil
		}
	}
	return config.ProviderInstance{}, fmt.Errorf("provider %q not found", providerID)
}
