// Package config loads ~/.vibeguard/config.yaml and resolves LLM settings
// with optional workspace policy overrides.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-1.0-config.md).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/alisaitteke/vibeguard/internal/paths"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

// ProviderInstance is a named LLM provider configuration (secrets live in credentials.yaml).
type ProviderInstance struct {
	ID       string `yaml:"id"`
	Driver   string `yaml:"driver"` // openai | anthropic | ollama | openai-compatible
	Model    string `yaml:"model"`
	BaseURL  string `yaml:"base_url,omitempty"`
	AuthMode string `yaml:"auth_mode"` // api_key | subscription
}

// AnalysisSettings configures on-demand command analysis.
type AnalysisSettings struct {
	Signature string `yaml:"signature"`
	Provider  string `yaml:"provider,omitempty"` // optional override; empty = default_provider
}

// LLMSettings holds resolved multi-provider LLM configuration.
type LLMSettings struct {
	Enabled         bool
	DefaultProvider string
	TimeoutMS       int
	Providers       []ProviderInstance
	Analysis        AnalysisSettings
}

// HistoryConfig holds command history retention settings for the local audit DB.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-5.0-history-cli.md).
type HistoryConfig struct {
	RetentionDays int
	MaxEvents     int
}

// UpdateConfig holds GitHub self-update settings for background checks and the CLI.
// See docs/plans/2026-07-02-1102-github-update/ (vgu-phase-3.0-update-cli.md).
type UpdateConfig struct {
	Enabled       bool
	CheckInterval string
	Channel       string
}

type fileDoc struct {
	LLM     llmFileBlock     `yaml:"llm"`
	History historyFileBlock `yaml:"history"`
	Update  updateFileBlock  `yaml:"update"`
}

type llmFileBlock struct {
	Enabled         *bool                  `yaml:"enabled,omitempty"`
	DefaultProvider string                 `yaml:"default_provider,omitempty"`
	TimeoutMS       int                    `yaml:"timeout_ms,omitempty"`
	Providers       []ProviderInstance     `yaml:"providers,omitempty"`
	Analysis        analysisSettingsBlock  `yaml:"analysis,omitempty"`
}

type analysisSettingsBlock struct {
	Signature string `yaml:"signature,omitempty"`
	Provider  string `yaml:"provider,omitempty"`
}

type historyFileBlock struct {
	RetentionDays *int `yaml:"retention_days,omitempty"`
	MaxEvents     *int `yaml:"max_events,omitempty"`
}

type updateFileBlock struct {
	Enabled       *bool  `yaml:"enabled,omitempty"`
	CheckInterval string `yaml:"check_interval,omitempty"`
	Channel       string `yaml:"channel,omitempty"`
}

func defaultLLMSettings() LLMSettings {
	return LLMSettings{
		Enabled:         false,
		DefaultProvider: "",
		TimeoutMS:       3000,
		Providers:       nil,
		Analysis: AnalysisSettings{
			Signature: "analysis",
			Provider:  "",
		},
	}
}

func defaultHistoryConfig() HistoryConfig {
	return HistoryConfig{
		RetentionDays: 30,
		MaxEvents:     50000,
	}
}

func defaultUpdateConfig() UpdateConfig {
	return UpdateConfig{
		Enabled:       true,
		CheckInterval: "6h",
		Channel:       "stable",
	}
}

// LoadUpdate reads self-update settings from ~/.vibeguard/config.yaml.
// Missing file or block uses defaults (enabled, 6h interval, stable channel).
func LoadUpdate() (UpdateConfig, error) {
	cfg := defaultUpdateConfig()

	configPath, err := paths.ConfigPath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config %s: %w", configPath, err)
	}

	var doc fileDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", configPath, err)
	}

	if doc.Update.Enabled != nil {
		cfg.Enabled = *doc.Update.Enabled
	}
	if doc.Update.CheckInterval != "" {
		cfg.CheckInterval = doc.Update.CheckInterval
	}
	if doc.Update.Channel != "" {
		cfg.Channel = doc.Update.Channel
	}
	return cfg, nil
}

// LoadHistory reads history retention settings from ~/.vibeguard/config.yaml.
// Missing file or block uses defaults (30 days, 50000 events). retention_days: 0
// disables time-based pruning; max_events: 0 disables count-based trimming.
func LoadHistory() (HistoryConfig, error) {
	cfg := defaultHistoryConfig()

	configPath, err := paths.ConfigPath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config %s: %w", configPath, err)
	}

	var doc fileDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", configPath, err)
	}

	if doc.History.RetentionDays != nil {
		cfg.RetentionDays = *doc.History.RetentionDays
	}
	if doc.History.MaxEvents != nil {
		cfg.MaxEvents = *doc.History.MaxEvents
	}
	return cfg, nil
}

// LoadLLMSettings reads global config.yaml and merges optional workspace policy llm.enabled.
// Missing config.yaml leaves LLM disabled (same as enabled: false).
func LoadLLMSettings(cwd string) (LLMSettings, error) {
	cfg := defaultLLMSettings()

	configPath, err := paths.ConfigPath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return mergeWorkspaceLLM(cfg, cwd)
		}
		return cfg, fmt.Errorf("read config %s: %w", configPath, err)
	}

	var doc fileDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", configPath, err)
	}

	cfg = applyFileBlock(cfg, doc.LLM)
	return mergeWorkspaceLLM(cfg, cwd)
}

func applyFileBlock(cfg LLMSettings, block llmFileBlock) LLMSettings {
	if block.Enabled != nil {
		cfg.Enabled = *block.Enabled
	}
	if block.DefaultProvider != "" {
		cfg.DefaultProvider = block.DefaultProvider
	}
	if block.TimeoutMS > 0 {
		cfg.TimeoutMS = block.TimeoutMS
	}
	if len(block.Providers) > 0 {
		cfg.Providers = append([]ProviderInstance(nil), block.Providers...)
	}
	if block.Analysis.Signature != "" {
		cfg.Analysis.Signature = block.Analysis.Signature
	}
	if block.Analysis.Provider != "" {
		cfg.Analysis.Provider = block.Analysis.Provider
	}
	return cfg
}

func settingsToFileBlock(settings LLMSettings) llmFileBlock {
	block := llmFileBlock{
		Enabled:         &settings.Enabled,
		DefaultProvider: settings.DefaultProvider,
		TimeoutMS:       settings.TimeoutMS,
		Providers:       append([]ProviderInstance(nil), settings.Providers...),
		Analysis: analysisSettingsBlock{
			Signature: settings.Analysis.Signature,
			Provider:  settings.Analysis.Provider,
		},
	}
	return block
}

func mergeWorkspaceLLM(cfg LLMSettings, cwd string) (LLMSettings, error) {
	if cwd == "" {
		return cfg, nil
	}

	workspacePath := filepath.Join(cwd, paths.DirName, paths.PolicyFile)
	data, err := os.ReadFile(workspacePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read workspace policy %s: %w", workspacePath, err)
	}

	var doc policy.File
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return cfg, fmt.Errorf("parse workspace policy %s: %w", workspacePath, err)
	}
	if doc.LLM != nil && doc.LLM.Enabled != nil {
		cfg.Enabled = *doc.LLM.Enabled
	}
	return cfg, nil
}

// DefaultConfigTemplate is written on install when no config file exists.
const DefaultConfigTemplate = `llm:
  enabled: false
  default_provider: ""
  timeout_ms: 3000
  providers: []
  # Example provider (uncomment and set default_provider after adding credentials):
  # providers:
  #   - id: my-openai
  #     driver: openai
  #     model: gpt-4o-mini
  #     base_url: ""
  #     auth_mode: api_key
  analysis:
    signature: analysis
    provider: ""

history:
  retention_days: 30
  max_events: 50000

update:
  enabled: true
  check_interval: 6h
  channel: stable
`

// EnsureDefault writes the default config template when the global file is missing.
func EnsureDefault() (string, error) {
	path, err := paths.ConfigPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(DefaultConfigTemplate), 0o600); err != nil {
		return "", err
	}
	return path, nil
}
