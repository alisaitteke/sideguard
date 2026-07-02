package llm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/alisaitteke/sideguard/internal/paths"
)

type signatureDoc struct {
	Name   string        `yaml:"name"`
	System string        `yaml:"system"`
	Rubric []rubricEntry `yaml:"rubric,omitempty"`
}

type rubricEntry struct {
	ID       string `yaml:"id"`
	Guidance string `yaml:"guidance"`
}

// DefaultAnalysisSignatureTemplate is the starter prompt for on-demand command analysis.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-1.0-config.md).
const DefaultAnalysisSignatureTemplate = `name: analysis
system: |
  You explain shell commands for a security-conscious user.
  Respond with JSON only: {"verdict":"safe|caution|dangerous|unknown","summary":"...","explanation":"..."}
`

// DefaultSignatureTemplate is the starter signature written on install.
const DefaultSignatureTemplate = `name: default
system: |
  You classify shell commands and MCP tool invocations for a local security daemon.
  Respond with JSON only: {"action":"allow|deny|ask","reason":"..."}
  Rules: uncertain → ask; destructive patterns (rm -rf, curl|sh, credential exfil) → deny unless clearly safe; read-only git/diagnostic → allow.
rubric:
  - id: destructive
    guidance: deny wget|bash, fork bombs, mass delete
  - id: ambiguous
    guidance: ask when intent unclear
`

// LoadSignature reads signatures/<name>.yaml and returns the system prompt body.
func LoadSignature(name string) (string, error) {
	dir, err := paths.SignaturesDir()
	if err != nil {
		return "", err
	}
	return loadSignatureFromDir(dir, name)
}

func loadSignatureFromDir(dir, name string) (string, error) {
	if name == "" {
		return "", errors.New("signature name required")
	}

	path := filepath.Join(dir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read signature %s: %w", path, err)
	}

	var doc signatureDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse signature %s: %w", path, err)
	}
	if doc.System == "" {
		return "", fmt.Errorf("signature %s: missing system field", path)
	}
	return doc.System, nil
}

// EnsureAnalysisSignature writes signatures/analysis.yaml when missing.
func EnsureAnalysisSignature() (string, error) {
	dir, err := paths.SignaturesDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "analysis.yaml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(DefaultAnalysisSignatureTemplate), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// EnsureDefaultSignature writes signatures/default.yaml when missing.
func EnsureDefaultSignature() (string, error) {
	dir, err := paths.SignaturesDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "default.yaml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(DefaultSignatureTemplate), 0o600); err != nil {
		return "", err
	}
	return path, nil
}
