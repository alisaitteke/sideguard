// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DefaultApprovalTimeout is how long a pending approval may wait before auto-deny.
const DefaultApprovalTimeout = 600 * time.Second

// ApprovalRecord is a persisted approval row.
type ApprovalRecord struct {
	ID           string
	Source       string
	Client       string
	Command      string
	CWD          string
	ToolName     string
	ToolInput    string
	Status       string
	Decision     string
	UserMessage  string
	AgentMessage string
	CreatedAt    time.Time
	DecidedAt    *time.Time
}

// DecisionResult is returned to long-poll waiters and hook clients.
type DecisionResult struct {
	Permission   string
	UserMessage  string
	AgentMessage string
}

type waiter chan DecisionResult

// CreateApproval inserts a new pending approval and records an audit event.
func (s *Store) CreateApproval(source, client, command, cwd, toolName string, toolInput map[string]any) (*ApprovalRecord, error) {
	id := uuid.NewString()
	now := time.Now().UTC()

	var toolInputJSON string
	if toolInput != nil {
		b, err := json.Marshal(toolInput)
		if err != nil {
			return nil, fmt.Errorf("marshal tool_input: %w", err)
		}
		toolInputJSON = string(b)
	}

	_, err := s.db.Exec(`
		INSERT INTO approvals (id, source, client, command, cwd, tool_name, tool_input, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
		id, source, client, command, cwd, nullString(toolName), nullString(toolInputJSON), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert approval: %w", err)
	}

	if err := s.writeAudit(id, "approval_requested", map[string]string{
		"source": source, "client": client, "command": command,
	}); err != nil {
		return nil, err
	}

	return &ApprovalRecord{
		ID: id, Source: source, Client: client, Command: command, CWD: cwd,
		ToolName: toolName, ToolInput: toolInputJSON, Status: "pending", CreatedAt: now,
	}, nil
}

// GetApproval loads a single approval by id.
func (s *Store) GetApproval(id string) (*ApprovalRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, source, client, command, cwd, tool_name, tool_input, status,
		       decision, user_message, agent_message, created_at, decided_at
		FROM approvals WHERE id = ?`, id)

	rec, err := scanApproval(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("approval %q not found", id)
		}
		return nil, err
	}
	return rec, nil
}

// ListPending returns all approvals with status pending.
func (s *Store) ListPending() ([]ApprovalRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, source, client, command, cwd, tool_name, tool_input, status,
		       decision, user_message, agent_message, created_at, decided_at
		FROM approvals WHERE status = 'pending'
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list pending: %w", err)
	}
	defer rows.Close()

	var out []ApprovalRecord
	for rows.Next() {
		rec, err := scanApproval(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

// DecideApproval records allow/deny, writes audit, and notifies waiters.
// Returns the decision result and whether this call changed state (false = idempotent replay).
func (s *Store) DecideApproval(id, decision, reason string) (*DecisionResult, bool, error) {
	rec, err := s.GetApproval(id)
	if err != nil {
		return nil, false, err
	}

	result := buildDecisionResult(decision, reason)

	if rec.Status != "pending" {
		existing := buildDecisionResult(rec.Decision, rec.UserMessage)
		if existing.Permission == result.Permission {
			return existing, false, nil
		}
		return nil, false, fmt.Errorf("approval %q already decided as %s", id, rec.Decision)
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(`
		UPDATE approvals
		SET status = 'decided', decision = ?, user_message = ?, agent_message = ?,
		    decided_at = ?
		WHERE id = ? AND status = 'pending'`,
		decision, result.UserMessage, result.AgentMessage, now.Format(time.RFC3339), id,
	)
	if err != nil {
		return nil, false, fmt.Errorf("update approval: %w", err)
	}

	if err := s.writeAudit(id, "approval_decided", map[string]string{
		"decision": decision, "reason": reason,
	}); err != nil {
		return nil, false, err
	}

	s.notifyWaiters(id, *result)
	return result, true, nil
}

// WaitForDecision blocks until the approval is decided, ctx expires, or the
// default approval timeout elapses (auto-deny with audit entry).
func (s *Store) WaitForDecision(ctx context.Context, id string) (*DecisionResult, error) {
	rec, err := s.GetApproval(id)
	if err != nil {
		return nil, err
	}

	if rec.Status != "pending" {
		return buildDecisionResult(rec.Decision, rec.UserMessage), nil
	}

	remaining := time.Until(rec.CreatedAt.Add(DefaultApprovalTimeout))
	if remaining <= 0 {
		return s.autoDenyExpired(id, "approval timed out before wait started")
	}

	waitCtx, cancel := context.WithTimeout(ctx, remaining)
	defer cancel()

	ch := make(waiter, 1)
	s.registerWaiter(id, ch)
	defer s.unregisterWaiter(id, ch)

	select {
	case result := <-ch:
		return &result, nil
	case <-waitCtx.Done():
		if waitCtx.Err() == context.DeadlineExceeded {
			return s.autoDenyExpired(id, "approval timed out")
		}
		return timeoutDenyResult(), nil
	}
}

// StartTimeoutSweeper periodically auto-denies expired pending approvals.
// Stops when ctx is cancelled. See vgf-phase-2.0-daemon-core.md.
func (s *Store) StartTimeoutSweeper(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepExpiredPending()
		}
	}
}

func (s *Store) sweepExpiredPending() {
	cutoff := time.Now().UTC().Add(-DefaultApprovalTimeout).Format(time.RFC3339)
	rows, err := s.db.Query(`
		SELECT id FROM approvals WHERE status = 'pending' AND created_at < ?`, cutoff)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		_, _ = s.autoDenyExpired(id, "approval timed out")
	}
}

func (s *Store) autoDenyExpired(id, message string) (*DecisionResult, error) {
	result, changed, err := s.DecideApproval(id, "deny", message)
	if err != nil {
		if changed {
			return nil, err
		}
		rec, getErr := s.GetApproval(id)
		if getErr != nil {
			return nil, err
		}
		return buildDecisionResult(rec.Decision, rec.UserMessage), nil
	}
	if changed {
		_ = s.writeAudit(id, "approval_auto_denied", map[string]string{"reason": message})
	}
	return result, nil
}

func (s *Store) registerWaiter(id string, ch waiter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waiters[id] = append(s.waiters[id], ch)
}

func (s *Store) unregisterWaiter(id string, ch waiter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.waiters[id]
	for i, w := range list {
		if w == ch {
			s.waiters[id] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(s.waiters[id]) == 0 {
		delete(s.waiters, id)
	}
}

func (s *Store) notifyWaiters(id string, result DecisionResult) {
	s.mu.Lock()
	list := append([]waiter(nil), s.waiters[id]...)
	s.mu.Unlock()

	for _, ch := range list {
		select {
		case ch <- result:
		default:
		}
	}
}

func (s *Store) writeAudit(approvalID, eventType string, payload map[string]string) error {
	var payloadJSON string
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal audit payload: %w", err)
		}
		payloadJSON = string(b)
	}
	_, err := s.db.Exec(`
		INSERT INTO audit_events (approval_id, event_type, payload, created_at)
		VALUES (?, ?, ?, ?)`,
		nullString(approvalID), eventType, nullString(payloadJSON), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func buildDecisionResult(decision, reason string) *DecisionResult {
	msg := reason
	if msg == "" {
		if decision == "allow" {
			msg = "approved by user"
		} else {
			msg = "denied by user"
		}
	}
	return &DecisionResult{
		Permission:   decision,
		UserMessage:  msg,
		AgentMessage: msg,
	}
}

func timeoutDenyResult() *DecisionResult {
	return &DecisionResult{
		Permission:   "deny",
		UserMessage:  "approval wait cancelled",
		AgentMessage: "approval wait cancelled",
	}
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type scannable interface {
	Scan(dest ...any) error
}

func scanApproval(row scannable) (*ApprovalRecord, error) {
	var rec ApprovalRecord
	var toolName, toolInput, decision, userMsg, agentMsg, createdAt, decidedAt sql.NullString

	err := row.Scan(
		&rec.ID, &rec.Source, &rec.Client, &rec.Command, &rec.CWD,
		&toolName, &toolInput, &rec.Status, &decision, &userMsg, &agentMsg,
		&createdAt, &decidedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan approval: %w", err)
	}
	rec.ToolName = toolName.String
	rec.ToolInput = toolInput.String
	rec.Decision = decision.String
	rec.UserMessage = userMsg.String
	rec.AgentMessage = agentMsg.String

	if createdAt.Valid {
		t, err := time.Parse(time.RFC3339, createdAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		rec.CreatedAt = t
	}

	if decidedAt.Valid {
		t, err := time.Parse(time.RFC3339, decidedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse decided_at: %w", err)
		}
		rec.DecidedAt = &t
	}
	return &rec, nil
}
