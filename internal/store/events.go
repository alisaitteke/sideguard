// Command event persistence for intercept audit history.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-4.0-history-store.md).
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alisaitteke/vibeguard/internal/llm"
	"github.com/google/uuid"
)

const (
	maxCommandNormLen     = 4 << 10  // 4KB
	maxCommandRedactedLen = 16 << 10 // 16KB

	schemaV3 = `
CREATE TABLE IF NOT EXISTS command_events (
	id TEXT PRIMARY KEY,
	created_at TEXT NOT NULL,
	source TEXT NOT NULL,
	client TEXT NOT NULL,
	cwd TEXT NOT NULL DEFAULT '',
	command_redacted TEXT NOT NULL DEFAULT '',
	command_norm TEXT NOT NULL DEFAULT '',
	tool_name TEXT,
	yaml_action TEXT NOT NULL DEFAULT '',
	detect_action TEXT NOT NULL DEFAULT '',
	detect_rules TEXT NOT NULL DEFAULT '[]',
	detect_score INTEGER NOT NULL DEFAULT 0,
	final_action TEXT NOT NULL,
	decision_by TEXT NOT NULL,
	reason TEXT NOT NULL DEFAULT '',
	approval_id TEXT,
	latency_ms INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_command_events_created_at ON command_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_command_events_cwd ON command_events(cwd);
CREATE INDEX IF NOT EXISTS idx_command_events_final_action ON command_events(final_action);
`
)

// CommandEvent is a persisted intercept decision row.
type CommandEvent struct {
	ID              string
	CreatedAt       time.Time
	Source          string
	Client          string
	CWD             string
	CommandRedacted string
	CommandNorm     string
	ToolName        string
	YamlAction      string
	DetectAction    string
	DetectRules     []string
	DetectScore     int
	FinalAction     string
	DecisionBy      string
	Reason          string
	ApprovalID      string
	LatencyMS       int64
}

// EventQuery filters command_events for GET /v1/events.
type EventQuery struct {
	Since  time.Time
	Denied bool
	CWD    string
	Limit  int
	Search string
}

// IngestEvent inserts a command event after redacting and truncating command fields.
func (s *Store) IngestEvent(e CommandEvent) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}

	e.CommandRedacted = truncate(llm.RedactCommand(e.CommandRedacted), maxCommandRedactedLen)
	e.CommandNorm = truncate(llm.RedactCommand(e.CommandNorm), maxCommandNormLen)

	rulesJSON, err := json.Marshal(e.DetectRules)
	if err != nil {
		return fmt.Errorf("marshal detect_rules: %w", err)
	}
	if len(e.DetectRules) == 0 {
		rulesJSON = []byte("[]")
	}

	_, err = s.db.Exec(`
		INSERT INTO command_events (
			id, created_at, source, client, cwd, command_redacted, command_norm,
			tool_name, yaml_action, detect_action, detect_rules, detect_score,
			final_action, decision_by, reason, approval_id, latency_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID,
		e.CreatedAt.UTC().Format(time.RFC3339),
		e.Source,
		e.Client,
		e.CWD,
		e.CommandRedacted,
		e.CommandNorm,
		nullString(e.ToolName),
		e.YamlAction,
		e.DetectAction,
		string(rulesJSON),
		e.DetectScore,
		e.FinalAction,
		e.DecisionBy,
		e.Reason,
		nullString(e.ApprovalID),
		e.LatencyMS,
	)
	if err != nil {
		return fmt.Errorf("insert command event: %w", err)
	}
	return nil
}

// QueryEvents returns command events matching the query, newest first.
func (s *Store) QueryEvents(q EventQuery) ([]CommandEvent, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	var (
		args    []any
		clauses []string
	)

	if !q.Since.IsZero() {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, q.Since.UTC().Format(time.RFC3339))
	}
	if q.Denied {
		clauses = append(clauses, "final_action = ?")
		args = append(args, "deny")
	}
	if q.CWD != "" {
		clauses = append(clauses, "cwd LIKE ?")
		args = append(args, q.CWD+"%")
	}
	if search := strings.TrimSpace(q.Search); search != "" {
		clauses = append(clauses, "(command_redacted LIKE ? OR command_norm LIKE ?)")
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
	}

	query := `
		SELECT id, created_at, source, client, cwd, command_redacted, command_norm,
		       tool_name, yaml_action, detect_action, detect_rules, detect_score,
		       final_action, decision_by, reason, approval_id, latency_ms
		FROM command_events`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query command events: %w", err)
	}
	defer rows.Close()

	var out []CommandEvent
	for rows.Next() {
		ev, err := scanCommandEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ev)
	}
	return out, rows.Err()
}

// PruneEvents removes command history rows older than retentionDays and/or trims
// the table to maxEvents (newest kept). retentionDays 0 skips time-based delete;
// maxEvents 0 skips count-based trim.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-5.0-history-cli.md).
func (s *Store) PruneEvents(retentionDays, maxEvents int) error {
	if retentionDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
		if _, err := s.db.Exec(
			`DELETE FROM command_events WHERE created_at < ?`,
			cutoff.Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("prune events by age: %w", err)
		}
	}

	if maxEvents <= 0 {
		return nil
	}

	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM command_events`).Scan(&count); err != nil {
		return fmt.Errorf("count command events: %w", err)
	}
	if count <= maxEvents {
		return nil
	}

	excess := count - maxEvents
	if _, err := s.db.Exec(`
		DELETE FROM command_events WHERE id IN (
			SELECT id FROM command_events ORDER BY created_at ASC LIMIT ?
		)`, excess); err != nil {
		return fmt.Errorf("prune events by count: %w", err)
	}
	return nil
}

func scanCommandEvent(row scannable) (*CommandEvent, error) {
	var ev CommandEvent
	var toolName, approvalID, createdAt sql.NullString
	var rulesJSON string

	err := row.Scan(
		&ev.ID, &createdAt, &ev.Source, &ev.Client, &ev.CWD,
		&ev.CommandRedacted, &ev.CommandNorm, &toolName,
		&ev.YamlAction, &ev.DetectAction, &rulesJSON, &ev.DetectScore,
		&ev.FinalAction, &ev.DecisionBy, &ev.Reason, &approvalID, &ev.LatencyMS,
	)
	if err != nil {
		return nil, fmt.Errorf("scan command event: %w", err)
	}

	if createdAt.Valid {
		t, err := time.Parse(time.RFC3339, createdAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		ev.CreatedAt = t
	}
	ev.ToolName = toolName.String
	ev.ApprovalID = approvalID.String

	if rulesJSON != "" {
		if err := json.Unmarshal([]byte(rulesJSON), &ev.DetectRules); err != nil {
			return nil, fmt.Errorf("unmarshal detect_rules: %w", err)
		}
	}
	return &ev, nil
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
