//go:build darwin

// Settings snapshot load/save for the macOS tray popover (no AppKit / CGO).
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-5.0-tray-settings-darwin.md).
package darwin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/llm"
)

// DriverOptionJSON describes one registered LLM driver for the settings dropdown.
type DriverOptionJSON struct {
	Name            string   `json:"name"`
	Label           string   `json:"label"`
	SupportsBaseURL bool     `json:"supports_base_url"`
	AuthModes       []string `json:"auth_modes"`
}

// ProviderRowJSON is one editable provider instance row in the settings form.
type ProviderRowJSON struct {
	ID            string `json:"id"`
	Driver        string `json:"driver"`
	Model         string `json:"model"`
	BaseURL       string `json:"base_url"`
	APIKey        string `json:"api_key"`
	KeyConfigured bool   `json:"key_configured"`
	IsDefault     bool   `json:"is_default"`
}

// SettingsJSON is the ObjC bridge payload for the settings screen.
type SettingsJSON struct {
	Drivers         []DriverOptionJSON  `json:"drivers"`
	Providers       []ProviderRowJSON   `json:"providers"`
	DefaultProvider string              `json:"default_provider"`
}

// settingsSavePayload is the JSON shape sent from ObjC on Save.
type settingsSavePayload struct {
	Providers       []ProviderRowJSON `json:"providers"`
	DefaultProvider string            `json:"default_provider"`
}

// LoadSettingsSnapshot builds the settings form state from config files and driver registry.
func LoadSettingsSnapshot() (SettingsJSON, error) {
	settings, err := config.LoadLLMSettings("")
	if err != nil {
		return SettingsJSON{}, err
	}

	drivers := llm.RegisteredDrivers()
	driverOpts := make([]DriverOptionJSON, 0, len(drivers))
	for _, d := range drivers {
		driverOpts = append(driverOpts, DriverOptionJSON{
			Name:            d.Name,
			Label:           d.Label,
			SupportsBaseURL: d.SupportsBaseURL,
			AuthModes:       append([]string(nil), d.AuthModes...),
		})
	}

	rows := make([]ProviderRowJSON, 0, len(settings.Providers))
	for _, p := range settings.Providers {
		masked, configured, statusErr := config.ProviderStatus(p.ID)
		if statusErr != nil {
			return SettingsJSON{}, statusErr
		}
		rows = append(rows, ProviderRowJSON{
			ID:            p.ID,
			Driver:        p.Driver,
			Model:         p.Model,
			BaseURL:       p.BaseURL,
			APIKey:        masked,
			KeyConfigured: configured,
			IsDefault:     p.ID == settings.DefaultProvider,
		})
	}

	return SettingsJSON{
		Drivers:         driverOpts,
		Providers:       rows,
		DefaultProvider: settings.DefaultProvider,
	}, nil
}

// SaveSettingsFromJSON validates and persists provider rows from the tray settings form.
// Blank or masked api_key values preserve the existing credential for that id.
func SaveSettingsFromJSON(payload string) error {
	var save settingsSavePayload
	if err := json.Unmarshal([]byte(payload), &save); err != nil {
		return fmt.Errorf("parse settings: %w", err)
	}

	current, err := config.LoadLLMSettings("")
	if err != nil {
		return err
	}

	newIDs := make(map[string]struct{}, len(save.Providers))
	instances := make([]config.ProviderInstance, 0, len(save.Providers))
	defaultID := save.DefaultProvider

	for _, row := range save.Providers {
		id := strings.TrimSpace(row.ID)
		if id == "" {
			return fmt.Errorf("provider id required")
		}
		if _, dup := newIDs[id]; dup {
			return fmt.Errorf("duplicate provider id %q", id)
		}
		newIDs[id] = struct{}{}

		driver := strings.TrimSpace(row.Driver)
		if driver == "" {
			return fmt.Errorf("provider %q: driver required", id)
		}

		if row.IsDefault {
			defaultID = id
		}

		instances = append(instances, config.ProviderInstance{
			ID:       id,
			Driver:   driver,
			Model:    strings.TrimSpace(row.Model),
			BaseURL:  strings.TrimSpace(row.BaseURL),
			AuthMode: "api_key",
		})
	}

	if defaultID != "" {
		if _, ok := newIDs[defaultID]; !ok {
			return fmt.Errorf("default_provider %q not found in providers", defaultID)
		}
	}

	for _, old := range current.Providers {
		if _, kept := newIDs[old.ID]; !kept {
			if err := config.RemoveProvider(old.ID); err != nil {
				return err
			}
		}
	}

	updated := current
	updated.Providers = instances
	updated.DefaultProvider = defaultID

	if err := config.SaveProviders(updated); err != nil {
		return err
	}

	for _, row := range save.Providers {
		id := strings.TrimSpace(row.ID)
		key := strings.TrimSpace(row.APIKey)
		if key == "" || isMaskedAPIKey(key) {
			continue
		}
		if err := config.SetProviderKey(id, key); err != nil {
			return err
		}
	}

	return nil
}

func isMaskedAPIKey(key string) bool {
	return strings.Contains(key, "…")
}
