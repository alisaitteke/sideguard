// Package config loads ~/.vibeguard/config.yaml and resolves LLM settings
// with optional workspace policy overrides.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-1.0-contracts.md).
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

// LLMConfig holds resolved LLM classification settings.
type LLMConfig struct {
	Enabled   bool
	Provider  string
	Model     string
	TimeoutMS int
	BaseURL   string
	Signature string
}

type fileDoc struct {
	LLM llmFileBlock `yaml:"llm"`
}

type llmFileBlock struct {
	Enabled   *bool  `yaml:"enabled,omitempty"`
	Provider  string `yaml:"provider,omitempty"`
	Model     string `yaml:"model,omitempty"`
	TimeoutMS int    `yaml:"timeout_ms,omitempty"`
	BaseURL   string `yaml:"base_url,omitempty"`
	Signature string `yaml:"signature,omitempty"`
}

func defaultLLMConfig() LLMConfig {
	return LLMConfig{
		Enabled:   false,
		Provider:  "openai",
		Model:     "gpt-4o-mini",
		TimeoutMS: 3000,
		Signature: "default",
	}
}

// Load reads global config.yaml and merges optional workspace policy llm.enabled.
// Missing config.yaml leaves LLM disabled (same as enabled: false).
func Load(cwd string) (LLMConfig, error) {
	cfg := defaultLLMConfig()

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

func applyFileBlock(cfg LLMConfig, block llmFileBlock) LLMConfig {
	if block.Enabled != nil {
		cfg.Enabled = *block.Enabled
	}
	if block.Provider != "" {
		cfg.Provider = block.Provider
	}
	if block.Model != "" {
		cfg.Model = block.Model
	}
	if block.TimeoutMS > 0 {
		cfg.TimeoutMS = block.TimeoutMS
	}
	if block.BaseURL != "" {
		cfg.BaseURL = block.BaseURL
	}
	if block.Signature != "" {
		cfg.Signature = block.Signature
	}
	return cfg
}

func mergeWorkspaceLLM(cfg LLMConfig, cwd string) (LLMConfig, error) {
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
  provider: openai
  model: gpt-4o-mini
  timeout_ms: 3000
  base_url: ""
  signature: default
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
