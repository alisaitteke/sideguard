package rules

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// WriteDefaults copies embedded detect rule packs into dir when each file is
// missing. Existing user files are never overwritten (idempotent bootstrap).
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-5.0-history-cli.md).
func WriteDefaults(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create rules dir: %w", err)
	}

	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		return fmt.Errorf("read embedded rules: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		dest := filepath.Join(dir, e.Name())
		if _, err := os.Stat(dest); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat %s: %w", dest, err)
		}

		data, err := FS.ReadFile(e.Name())
		if err != nil {
			return fmt.Errorf("read embedded rule %s: %w", e.Name(), err)
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return fmt.Errorf("write rule %s: %w", dest, err)
		}
	}
	return nil
}
