// Package store provides SQLite persistence for approvals and audit events.
// See docs/plans/2026-07-01-0127-sideguard-foundation/ (vgf-phase-1.0-project-init.md).
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

const schemaV1 = `
CREATE TABLE IF NOT EXISTS approvals (
	id TEXT PRIMARY KEY,
	source TEXT NOT NULL,
	client TEXT NOT NULL,
	command TEXT NOT NULL DEFAULT '',
	cwd TEXT NOT NULL DEFAULT '',
	tool_name TEXT,
	tool_input TEXT,
	status TEXT NOT NULL DEFAULT 'pending',
	decision TEXT,
	user_message TEXT,
	agent_message TEXT,
	created_at TEXT NOT NULL,
	decided_at TEXT
);

CREATE TABLE IF NOT EXISTS audit_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	approval_id TEXT,
	event_type TEXT NOT NULL,
	payload TEXT,
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approvals_status ON approvals(status);
CREATE INDEX IF NOT EXISTS idx_audit_events_approval_id ON audit_events(approval_id);
`

// Store wraps the SQLite database for approvals and audit events.
type Store struct {
	db      *sql.DB
	mu      sync.Mutex
	waiters map[string][]waiter
}

// Open opens (or creates) the SQLite database at path and applies migrations.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	s := &Store{db: db, waiters: make(map[string][]waiter)}
	if err := s.Migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Migrate applies schema migrations.
func (s *Store) Migrate() error {
	if _, err := s.db.Exec(schemaV1); err != nil {
		return fmt.Errorf("migrate v1: %w", err)
	}
	if _, err := s.db.Exec(schemaV2); err != nil {
		return fmt.Errorf("migrate v2: %w", err)
	}
	if _, err := s.db.Exec(schemaV3); err != nil {
		return fmt.Errorf("migrate v3: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB exposes the underlying connection for Phase 2 handlers.
func (s *Store) DB() *sql.DB {
	return s.db
}
