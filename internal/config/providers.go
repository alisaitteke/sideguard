// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Provider instance helpers for multi-provider LLM settings.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-1.0-config.md).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/alisaitteke/sideguard/internal/paths"
)

var (
	validDrivers   = []string{"openai", "anthropic", "ollama", "openai-compatible"}
	validAuthModes = []string{"api_key", "subscription"}
)

// ValidateProviderInstance checks driver, auth_mode, and non-empty id.
func ValidateProviderInstance(p ProviderInstance) error {
	if p.ID == "" {
		return errors.New("provider id required")
	}
	if !slices.Contains(validDrivers, p.Driver) {
		return fmt.Errorf("unknown driver %q (want one of %v)", p.Driver, validDrivers)
	}
	if !slices.Contains(validAuthModes, p.AuthMode) {
		return fmt.Errorf("unknown auth_mode %q (want one of %v)", p.AuthMode, validAuthModes)
	}
	return nil
}

// ValidateLLMSettings checks provider list integrity and default_provider reference.
func ValidateLLMSettings(settings LLMSettings) error {
	seen := make(map[string]struct{}, len(settings.Providers))
	for _, p := range settings.Providers {
		if err := ValidateProviderInstance(p); err != nil {
			return fmt.Errorf("provider %q: %w", p.ID, err)
		}
		if _, dup := seen[p.ID]; dup {
			return fmt.Errorf("duplicate provider id %q", p.ID)
		}
		seen[p.ID] = struct{}{}
	}
	if settings.DefaultProvider != "" {
		if _, ok := seen[settings.DefaultProvider]; !ok {
			return fmt.Errorf("default_provider %q not found in providers", settings.DefaultProvider)
		}
	}
	return nil
}

// LoadProviders returns provider instances from ~/.sideguard/config.yaml.
func LoadProviders() ([]ProviderInstance, error) {
	settings, err := LoadLLMSettings("")
	if err != nil {
		return nil, err
	}
	return slices.Clone(settings.Providers), nil
}

// SaveProviders writes LLM settings to config.yaml (mode 0600), preserving history/update blocks.
func SaveProviders(settings LLMSettings) error {
	if err := ValidateLLMSettings(settings); err != nil {
		return err
	}
	return writeLLMSettings(settings)
}

// AddProvider appends a provider instance and persists config.
func AddProvider(settings LLMSettings, p ProviderInstance) (LLMSettings, error) {
	if err := ValidateProviderInstance(p); err != nil {
		return settings, err
	}
	for _, existing := range settings.Providers {
		if existing.ID == p.ID {
			return settings, fmt.Errorf("duplicate provider id %q", p.ID)
		}
	}
	settings.Providers = append(settings.Providers, p)
	if err := SaveProviders(settings); err != nil {
		return settings, err
	}
	return settings, nil
}

// UpdateProvider replaces a provider instance by id and persists config.
func UpdateProvider(settings LLMSettings, p ProviderInstance) (LLMSettings, error) {
	if err := ValidateProviderInstance(p); err != nil {
		return settings, err
	}
	idx := -1
	for i, existing := range settings.Providers {
		if existing.ID == p.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return settings, fmt.Errorf("provider %q not found", p.ID)
	}
	settings.Providers[idx] = p
	if err := SaveProviders(settings); err != nil {
		return settings, err
	}
	return settings, nil
}

// SetDefaultProvider sets default_provider and persists config.
func SetDefaultProvider(id string) error {
	settings, err := LoadLLMSettings("")
	if err != nil {
		return err
	}
	if id != "" {
		found := false
		for _, p := range settings.Providers {
			if p.ID == id {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("provider %q not found", id)
		}
	}
	settings.DefaultProvider = id
	return SaveProviders(settings)
}

// RemoveProvider deletes a provider instance and its credentials entry.
func RemoveProvider(id string) error {
	settings, err := LoadLLMSettings("")
	if err != nil {
		return err
	}

	found := false
	filtered := settings.Providers[:0]
	for _, p := range settings.Providers {
		if p.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, p)
	}
	if !found {
		return fmt.Errorf("provider %q not found", id)
	}
	settings.Providers = filtered
	if settings.DefaultProvider == id {
		settings.DefaultProvider = ""
	}
	if err := SaveProviders(settings); err != nil {
		return err
	}
	return removeProviderCredential(id)
}

func readConfigDoc() (fileDoc, string, error) {
	var doc fileDoc
	configPath, err := paths.ConfigPath()
	if err != nil {
		return doc, "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doc, configPath, nil
		}
		return doc, configPath, fmt.Errorf("read config %s: %w", configPath, err)
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return doc, configPath, fmt.Errorf("parse config %s: %w", configPath, err)
	}
	return doc, configPath, nil
}

func writeLLMSettings(settings LLMSettings) error {
	doc, configPath, err := readConfigDoc()
	if err != nil {
		return err
	}
	doc.LLM = settingsToFileBlock(settings)
	return writeConfigDoc(configPath, doc)
}

func writeConfigDoc(configPath string, doc fileDoc) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return err
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, out, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", configPath, err)
	}
	return nil
}
