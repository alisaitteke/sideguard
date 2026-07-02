package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alisaitteke/vibeguard/internal/paths"
)

// FileStateStore persists update state at ~/.vibeguard/update-state.json.
type FileStateStore struct {
	path string
}

// NewFileStateStore returns the default on-disk state store.
func NewFileStateStore() (*FileStateStore, error) {
	p, err := paths.UpdateStatePath()
	if err != nil {
		return nil, err
	}
	return &FileStateStore{path: p}, nil
}

// NewFileStateStoreAt returns a state store at an explicit path (tests).
func NewFileStateStoreAt(path string) *FileStateStore {
	return &FileStateStore{path: path}
}

// Load reads update state; a missing file returns a zero State.
func (s *FileStateStore) Load() (State, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("read update state: %w", err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, fmt.Errorf("parse update state: %w", err)
	}
	return st, nil
}

// Save atomically writes update state via temp file + rename.
func (s *FileStateStore) Save(st State) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write update state: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("commit update state: %w", err)
	}
	return nil
}

// TouchLastCheck updates last_check_at and optionally latest_known.
func TouchLastCheck(store StateStore, latest string) error {
	st, err := store.Load()
	if err != nil {
		return err
	}
	st.LastCheckAt = time.Now().UTC()
	if latest != "" {
		st.LatestKnown = latest
	}
	return store.Save(st)
}
