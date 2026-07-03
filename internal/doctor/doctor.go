// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Package doctor diagnoses SideGuard install health and bypass vectors.
// See docs/plans/2026-07-01-0127-sideguard-foundation/ (vgf-phase-8.0-hardening.md).
package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alisaitteke/sideguard/internal/config"
	"github.com/alisaitteke/sideguard/internal/daemon"
	"github.com/alisaitteke/sideguard/internal/install"
	"github.com/alisaitteke/sideguard/internal/paths"
	"github.com/alisaitteke/sideguard/internal/policy"
)

// Severity classifies a doctor finding.
type Severity string

const (
	SeverityOK   Severity = "OK"
	SeverityWarn Severity = "WARN"
	SeverityHigh Severity = "HIGH"
)

// Finding is one diagnostic check result.
type Finding struct {
	Check    string
	Severity Severity
	Message  string
}

// Report aggregates all findings from a doctor run.
type Report struct {
	Findings []Finding
}

// HasHigh reports whether any finding is HIGH severity.
func (r Report) HasHigh() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityHigh {
			return true
		}
	}
	return false
}

// ExitCode returns 1 when HIGH findings exist, else 0.
func (r Report) ExitCode() int {
	if r.HasHigh() {
		return 1
	}
	return 0
}

// Options controls which clients are checked.
type Options struct {
	Cursor bool
	Claude bool
	Cwd    string
}

// Run executes all doctor checks and returns a structured report.
func Run(opts Options) (Report, error) {
	if !opts.Cursor && !opts.Claude {
		opts.Cursor = true
		opts.Claude = true
	}

	cwd := opts.Cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return Report{}, err
		}
	}

	var report Report
	report.Findings = append(report.Findings, checkDaemon()...)
	report.Findings = append(report.Findings, checkPolicy(cwd)...)
	report.Findings = append(report.Findings, checkLLMConfig(cwd)...)

	targets, err := install.Discover(install.DiscoverOptions{
		Cursor: opts.Cursor,
		Claude: opts.Claude,
		Cwd:    cwd,
	})
	if err != nil {
		return Report{}, err
	}

	for _, t := range targets {
		switch t.Kind {
		case install.KindHooks:
			report.Findings = append(report.Findings, checkHooks(t)...)
		case install.KindMCP:
			report.Findings = append(report.Findings, checkMCPWrap(t)...)
		}
	}

	return report, nil
}

func checkDaemon() []Finding {
	running, pid := daemon.IsRunning()
	if running {
		return []Finding{{
			Check:    "daemon",
			Severity: SeverityOK,
			Message:  fmt.Sprintf("daemon running (pid %d)", pid),
		}}
	}
	return []Finding{{
		Check:    "daemon",
		Severity: SeverityHigh,
		Message:  "daemon is not running — approvals cannot be queued",
	}}
}

func checkLLMConfig(cwd string) []Finding {
	cfg, err := config.LoadLLMSettings(cwd)
	if err != nil {
		return []Finding{{
			Check:    "llm_config",
			Severity: SeverityWarn,
			Message:  fmt.Sprintf("LLM config invalid: %v", err),
		}}
	}

	configPath, pathErr := paths.ConfigPath()
	if pathErr == nil {
		if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) && !cfg.Enabled {
			return []Finding{{
				Check:    "llm_config_missing",
				Severity: SeverityOK,
				Message:  "LLM disabled; config.yaml not present (using defaults)",
			}}
		}
	}

	if !cfg.Enabled {
		return nil
	}

	var findings []Finding

	creds, credErr := config.ResolveProviderCredentials()
	if credErr != nil {
		findings = append(findings, Finding{
			Check:    "llm_credentials",
			Severity: SeverityWarn,
			Message:  fmt.Sprintf("cannot read credentials: %v", credErr),
		})
		return findings
	}

	defaultID := cfg.DefaultProvider
	var defaultDriver string
	for _, p := range cfg.Providers {
		if p.ID == defaultID {
			defaultDriver = p.Driver
			break
		}
	}

	if defaultID == "" {
		findings = append(findings, Finding{
			Check:    "llm_enabled_no_default",
			Severity: SeverityWarn,
			Message:  "LLM enabled but no default_provider configured",
		})
	} else if !config.HasAPIKeyForProvider(defaultDriver, creds, defaultID) {
		findings = append(findings, Finding{
			Check:    "llm_enabled_no_credentials",
			Severity: SeverityWarn,
			Message:  fmt.Sprintf("LLM enabled with provider %q but no API key configured", defaultID),
		})
	}

	credPath, err := paths.CredentialsPath()
	if err == nil {
		if info, statErr := os.Stat(credPath); statErr == nil {
			if info.Mode().Perm() != 0o600 {
				findings = append(findings, Finding{
					Check:    "llm_credentials_perms",
					Severity: SeverityWarn,
					Message:  fmt.Sprintf("credentials.yaml should be mode 0600, got %04o", info.Mode().Perm()),
				})
			}
		} else if !os.IsNotExist(statErr) {
			findings = append(findings, Finding{
				Check:    "llm_credentials",
				Severity: SeverityWarn,
				Message:  fmt.Sprintf("cannot stat credentials: %v", statErr),
			})
		}
	}

	if len(findings) == 0 {
		findings = append(findings, Finding{
			Check:    "llm_config",
			Severity: SeverityOK,
			Message:  fmt.Sprintf("LLM enabled (default provider %q)", cfg.DefaultProvider),
		})
	}

	return findings
}

func checkPolicy(cwd string) []Finding {
	_, err := policy.Load(cwd)
	if err != nil {
		return []Finding{{
			Check:    "policy",
			Severity: SeverityHigh,
			Message:  fmt.Sprintf("policy invalid: %v", err),
		}}
	}
	return []Finding{{
		Check:    "policy",
		Severity: SeverityOK,
		Message:  "policy YAML valid",
	}}
}

func checkHooks(t install.Target) []Finding {
	data, err := os.ReadFile(t.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Finding{{
				Check:    fmt.Sprintf("hooks:%s", t.Client),
				Severity: SeverityOK,
				Message:  fmt.Sprintf("%s not configured (%s)", t.Client, shortenPath(t.Path)),
			}}
		}
		return []Finding{{
			Check:    fmt.Sprintf("hooks:%s", t.Client),
			Severity: SeverityWarn,
			Message:  fmt.Sprintf("cannot read %s: %v", t.Path, err),
		}}
	}

	hasShell, hasMCP := detectSideguardHooks(t.Client, data)
	var findings []Finding

	if !hasShell {
		findings = append(findings, Finding{
			Check:    fmt.Sprintf("hooks:%s:shell", t.Client),
			Severity: SeverityHigh,
			Message:  fmt.Sprintf("SideGuard shell hook missing in %s — shell commands may bypass approval", shortenPath(t.Path)),
		})
	} else {
		findings = append(findings, Finding{
			Check:    fmt.Sprintf("hooks:%s:shell", t.Client),
			Severity: SeverityOK,
			Message:  fmt.Sprintf("shell hook present (%s)", shortenPath(t.Path)),
		})
	}

	if !hasMCP {
		findings = append(findings, Finding{
			Check:    fmt.Sprintf("hooks:%s:mcp", t.Client),
			Severity: SeverityHigh,
			Message:  fmt.Sprintf("SideGuard MCP hook missing in %s — MCP tools may bypass approval", shortenPath(t.Path)),
		})
	} else {
		findings = append(findings, Finding{
			Check:    fmt.Sprintf("hooks:%s:mcp", t.Client),
			Severity: SeverityOK,
			Message:  fmt.Sprintf("MCP hook present (%s)", shortenPath(t.Path)),
		})
	}
	return findings
}

func checkMCPWrap(t install.Target) []Finding {
	data, err := os.ReadFile(t.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{
			Check:    fmt.Sprintf("mcp:%s", t.Client),
			Severity: SeverityWarn,
			Message:  fmt.Sprintf("cannot read %s: %v", t.Path, err),
		}}
	}

	unwrapped := findUnwrappedStdioServers(data)
	if len(unwrapped) == 0 {
		return []Finding{{
			Check:    fmt.Sprintf("mcp:%s", t.Client),
			Severity: SeverityOK,
			Message:  fmt.Sprintf("stdio MCP servers wrapped or none (%s)", shortenPath(t.Path)),
		}}
	}
	return []Finding{{
		Check:    fmt.Sprintf("mcp:%s", t.Client),
		Severity: SeverityWarn,
		Message:  fmt.Sprintf("unwrapped stdio MCP servers in %s: %s — direct bypass possible",
			shortenPath(t.Path), strings.Join(unwrapped, ", ")),
	}}
}

func detectSideguardHooks(client install.Client, data []byte) (shell bool, mcp bool) {
	text := string(data)
	switch client {
	case install.ClientCursor, install.ClientClaude:
		shell = strings.Contains(text, "hook shell") || strings.Contains(text, "sideguard hook shell")
		mcp = strings.Contains(text, "hook mcp") || strings.Contains(text, "sideguard hook mcp")
	}
	return shell, mcp
}

type mcpInspectDoc struct {
	MCPServers map[string]mcpInspectEntry `json:"mcpServers"`
}

type mcpInspectEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	URL     string   `json:"url"`
}

func findUnwrappedStdioServers(data []byte) []string {
	var doc mcpInspectDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	if len(doc.MCPServers) == 0 {
		return nil
	}

	var names []string
	for name, entry := range doc.MCPServers {
		if entry.URL != "" || entry.Command == "" {
			continue
		}
		if isWrappedEntry(entry) {
			continue
		}
		names = append(names, name)
	}
	return names
}

func isWrappedEntry(entry mcpInspectEntry) bool {
	cmd := filepath.Base(entry.Command)
	if cmd != "sideguard" && !strings.HasSuffix(entry.Command, "/sideguard") {
		return false
	}
	return len(entry.Args) >= 2 && entry.Args[0] == "wrap" && entry.Args[1] == "--"
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
