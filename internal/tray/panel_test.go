package tray

import (
	"encoding/json"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

// panelJSON mirrors darwin.PanelJSON for contract tests (no CGO required).
type panelJSON struct {
	DaemonStatus string          `json:"daemon_status"`
	PendingCount string          `json:"pending_count"`
	ModeIndex    int             `json:"mode_index"`
	ModeEnabled  bool            `json:"mode_enabled"`
	Rows         []panelJSONRow  `json:"rows"`
	OverflowHint string          `json:"overflow_hint"`
	EmptyMessage string          `json:"empty_message"`
}

type panelJSONRow struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func TestBuildPanelRows_CapAtTen(t *testing.T) {
	t.Parallel()

	items := make([]api.PendingApproval, 12)
	for i := range items {
		items[i].ID = "id"
		items[i].Client = "cursor"
	}

	content := BuildPanelRows(PanelSnapshot{
		Items:    items,
		Mode:     approvalmode.Ask,
		HealthOK: true,
	})

	if len(content.Rows) != maxVisiblePending {
		t.Fatalf("rows = %d, want %d", len(content.Rows), maxVisiblePending)
	}
	if content.OverflowHint == "" {
		t.Fatal("expected overflow hint")
	}
	if content.EmptyMessage != "" {
		t.Fatalf("empty message should be unset, got %q", content.EmptyMessage)
	}
}

func TestBuildPanelRows_OverflowLabel(t *testing.T) {
	t.Parallel()

	items := make([]api.PendingApproval, 11)
	content := BuildPanelRows(PanelSnapshot{
		Items:    items,
		Mode:     approvalmode.Ask,
		HealthOK: true,
	})

	want := overflowLabel(1)
	if content.OverflowHint != want {
		t.Fatalf("overflow hint = %q, want %q", content.OverflowHint, want)
	}
}

func TestBuildPanelRows_PendingItemsProduceRows(t *testing.T) {
	t.Parallel()

	items := []api.PendingApproval{{
		ID:         "d476b56e-91a7-4595-9c9c-63d4fb960806",
		Client:     "cursor",
		Command:    "git status",
		AgeSeconds: 54,
	}}

	content := BuildPanelRows(PanelSnapshot{
		Items:    items,
		Mode:     approvalmode.Ask,
		HealthOK: true,
	})

	if len(content.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(content.Rows))
	}
	if content.Rows[0].ID != items[0].ID {
		t.Fatalf("row ID = %q, want %q", content.Rows[0].ID, items[0].ID)
	}
	if content.Rows[0].Label == "" {
		t.Fatal("row label should not be empty")
	}
	if content.PendingCount != "● 1 pending" {
		t.Fatalf("pending count = %q, want %q", content.PendingCount, "● 1 pending")
	}
}

func TestBuildPanelRows_JSONPayloadMatchesObjCContract(t *testing.T) {
	t.Parallel()

	content := BuildPanelRows(PanelSnapshot{
		Items: []api.PendingApproval{{
			ID:         "abc-def-123",
			Client:     "cursor",
			Command:    "echo hi",
			AgeSeconds: 5,
		}},
		Mode:     approvalmode.Ask,
		HealthOK: true,
	})

	rows := make([]panelJSONRow, 0, len(content.Rows))
	for _, row := range content.Rows {
		rows = append(rows, panelJSONRow{ID: row.ID, Label: row.Label})
	}
	payload := panelJSON{
		DaemonStatus: content.DaemonStatus,
		PendingCount: content.PendingCount,
		ModeIndex:    content.ModeIndex,
		ModeEnabled:  content.ModeEnabled,
		Rows:         rows,
		OverflowHint: content.OverflowHint,
		EmptyMessage: content.EmptyMessage,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	rawRows, ok := decoded["rows"].([]any)
	if !ok || len(rawRows) != 1 {
		t.Fatalf("rows = %#v, want one-element array", decoded["rows"])
	}
	row, ok := rawRows[0].(map[string]any)
	if !ok {
		t.Fatalf("row type = %T", rawRows[0])
	}
	if row["id"] != "abc-def-123" {
		t.Fatalf("row id = %#v", row["id"])
	}
	if row["label"] == "" || row["label"] == nil {
		t.Fatalf("row label missing: %#v", row["label"])
	}
}

func TestBuildPanelRows_EmptyState(t *testing.T) {
	t.Parallel()

	content := BuildPanelRows(PanelSnapshot{
		Items:    nil,
		Mode:     approvalmode.Ask,
		HealthOK: true,
	})

	if len(content.Rows) != 0 {
		t.Fatalf("rows = %d, want 0", len(content.Rows))
	}
	if content.EmptyMessage != "No pending approvals" {
		t.Fatalf("empty message = %q", content.EmptyMessage)
	}
	if content.OverflowHint != "" {
		t.Fatalf("overflow hint = %q, want empty", content.OverflowHint)
	}
}

func TestBuildPanelRows_DaemonDown(t *testing.T) {
	t.Parallel()

	content := BuildPanelRows(PanelSnapshot{
		Items:    []api.PendingApproval{{ID: "x"}},
		HealthOK: false,
		Err:      errDaemonUnreachable(),
	})

	if len(content.Rows) != 0 {
		t.Fatalf("rows = %d, want 0 when daemon down", len(content.Rows))
	}
	if content.EmptyMessage != "" {
		t.Fatalf("empty message = %q, want unset when down", content.EmptyMessage)
	}
	if content.ModeEnabled {
		t.Fatal("mode control should be disabled when daemon down")
	}
}

func TestModeFromSegmentIndex(t *testing.T) {
	t.Parallel()

	if got := ModeFromSegmentIndex(0); got != approvalmode.Ask {
		t.Fatalf("0: %v", got)
	}
	if got := ModeFromSegmentIndex(1); got != approvalmode.AutoAllow {
		t.Fatalf("1: %v", got)
	}
	if got := ModeFromSegmentIndex(2); got != approvalmode.AutoDeny {
		t.Fatalf("2: %v", got)
	}
}
