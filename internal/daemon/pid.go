// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func writePID(path string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create pid dir: %w", err)
	}
	data := fmt.Sprintf("%d\n", pid)
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}
	return nil
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid pid file: %w", err)
	}
	return pid, nil
}

func removePID(path string) {
	_ = os.Remove(path)
}
