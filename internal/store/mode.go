// Approval mode persistence and auto-decide engine for the daemon store.
// See docs/plans/2026-07-01-1515-global-approval-mode/ (gam-phase-1.0-store-mode.md).
package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

const schemaV2 = `
CREATE TABLE IF NOT EXISTS daemon_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
INSERT OR IGNORE INTO daemon_settings (key, value) VALUES ('approval_mode', 'ask');
`

// GetApprovalMode returns the persisted global approval mode (default ask).
func (s *Store) GetApprovalMode() (approvalmode.Mode, error) {
	var value string
	err := s.db.QueryRow(`
		SELECT value FROM daemon_settings WHERE key = ?`, approvalmode.SettingKey).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return approvalmode.Ask, nil
		}
		return "", fmt.Errorf("get approval mode: %w", err)
	}
	return approvalmode.Parse(value)
}

// SetApprovalMode persists the mode and sweeps all pending approvals with the new auto decision.
func (s *Store) SetApprovalMode(m approvalmode.Mode) error {
	if !m.Valid() {
		return fmt.Errorf("invalid approval mode %q", m)
	}

	if _, err := s.db.Exec(`
		INSERT INTO daemon_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		approvalmode.SettingKey, string(m),
	); err != nil {
		return fmt.Errorf("set approval mode: %w", err)
	}

	if err := s.writeAudit("", "approval_mode_changed", map[string]string{
		"mode": string(m),
	}); err != nil {
		return err
	}

	decision := m.Decision()
	if decision == "" {
		return nil
	}

	pending, err := s.ListPending()
	if err != nil {
		return err
	}
	reason := m.AutoReason()
	for _, rec := range pending {
		if _, _, err := s.DecideApproval(rec.ID, decision, reason); err != nil {
			return err
		}
		if err := s.writeAudit(rec.ID, "approval_auto_decided", map[string]string{
			"mode": string(m),
		}); err != nil {
			return err
		}
	}
	return nil
}

// MaybeAutoDecide applies the current mode to a newly created approval.
// Returns the decision result and whether an auto decision was made.
func (s *Store) MaybeAutoDecide(id string) (*DecisionResult, bool, error) {
	mode, err := s.GetApprovalMode()
	if err != nil {
		return nil, false, err
	}
	decision := mode.Decision()
	if decision == "" {
		return nil, false, nil
	}

	reason := mode.AutoReason()
	result, changed, err := s.DecideApproval(id, decision, reason)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return result, true, nil
	}
	if err := s.writeAudit(id, "approval_auto_decided", map[string]string{
		"mode": string(mode),
	}); err != nil {
		return nil, false, err
	}
	return result, true, nil
}
