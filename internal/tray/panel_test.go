package tray

import (
	"encoding/json"
	"testing"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
)

// panelJSON mirrors darwin.PanelJSON for contract tests (no CGO required).
type panelJSON struct {
	ModeIndex       int            `json:"mode_index"`
	ModeEnabled     bool           `json:"mode_enabled"`
	PendingRows     []panelJSONRow `json:"pending_rows"`
	HistoryRows     []panelJSONRow `json:"history_rows"`
	HistoryHasMore  bool           `json:"history_has_more"`
	PendingOverflow string         `json:"pending_overflow"`
	EmptyMessage    string         `json:"empty_message"`
	FooterDaemon    string         `json:"footer_daemon"`
	FooterPending   string         `json:"footer_pending"`
	UpdateVisible   bool           `json:"update_visible"`
	UpdateLabel     string         `json:"update_label"`
	UpdateEnabled   bool           `json:"update_enabled"`
}

type panelJSONRow struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
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
	if content.Rows[0].Detail != "git status" {
		t.Fatalf("row detail = %q, want git status", content.Rows[0].Detail)
	}
	if content.PendingCount != "● 1 pending" {
		t.Fatalf("pending count = %q, want %q", content.PendingCount, "● 1 pending")
	}
}

func TestBuildPanelRows_UpdateFooter(t *testing.T) {
	t.Parallel()

	content := BuildPanelRows(PanelSnapshot{
		Items:    nil,
		Mode:     approvalmode.Ask,
		HealthOK: true,
		Update: UpdateUIState{
			Available: true,
			Version:   "2.0.0",
			Label:     "Update available: v2.0.0",
		},
	})

	if !content.UpdateVisible {
		t.Fatal("expected update footer visible")
	}
	if content.UpdateLabel != "Update available: v2.0.0" {
		t.Fatalf("label = %q", content.UpdateLabel)
	}
	if !content.UpdateEnabled {
		t.Fatal("expected install enabled")
	}
}

func TestBuildPanelRows_UpdateInstallingDisabled(t *testing.T) {
	t.Parallel()

	content := BuildPanelRows(PanelSnapshot{
		HealthOK: true,
		Update: UpdateUIState{
			Available:  true,
			Version:    "2.0.0",
			Label:      "Update available: v2.0.0",
			Installing: true,
		},
	})

	if content.UpdateEnabled {
		t.Fatal("expected install disabled while installing")
	}
}

func TestBuildTrayContent_DarwinJSONContract(t *testing.T) {
	t.Parallel()

	content := BuildTrayContent(PanelSnapshot{
		Items: []api.PendingApproval{{
			ID:         "abc-def-123",
			Client:     "cursor",
			Command:    "echo hi",
			AgeSeconds: 5,
		}},
		History: []api.CommandEvent{{
			ID:              "hist-1",
			ApprovalID:      "resolved-1",
			Client:          "claude",
			FinalAction:     "deny",
			CommandRedacted: "rm file",
			CreatedAt:       "2026-07-02T10:00:00Z",
		}},
		HistoryHasMore: true,
		Mode:           approvalmode.Ask,
		HealthOK:       true,
	})

	pendingRows := make([]panelJSONRow, 0, len(content.PendingRows))
	for _, row := range content.PendingRows {
		pendingRows = append(pendingRows, panelJSONRow{
			Kind:   string(row.Kind),
			ID:     row.ID,
			Label:  row.Label,
			Detail: row.Detail,
		})
	}
	historyRows := make([]panelJSONRow, 0, len(content.HistoryRows))
	for _, row := range content.HistoryRows {
		historyRows = append(historyRows, panelJSONRow{
			Kind:   string(row.Kind),
			ID:     row.ID,
			Label:  row.Label,
			Detail: row.Detail,
		})
	}

	payload := panelJSON{
		ModeIndex:      content.ModeIndex,
		ModeEnabled:    content.ModeEnabled,
		PendingRows:    pendingRows,
		HistoryRows:    historyRows,
		HistoryHasMore: content.HistoryHasMore,
		FooterDaemon:   content.FooterDaemon,
		FooterPending:  content.FooterPending,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	rawPending, ok := decoded["pending_rows"].([]any)
	if !ok || len(rawPending) != 1 {
		t.Fatalf("pending_rows = %#v, want one-element array", decoded["pending_rows"])
	}
	pending, ok := rawPending[0].(map[string]any)
	if !ok {
		t.Fatalf("pending row type = %T", rawPending[0])
	}
	if pending["id"] != "abc-def-123" {
		t.Fatalf("pending id = %#v", pending["id"])
	}
	if pending["kind"] != "pending" {
		t.Fatalf("pending kind = %#v", pending["kind"])
	}

	rawHistory, ok := decoded["history_rows"].([]any)
	if !ok || len(rawHistory) != 1 {
		t.Fatalf("history_rows = %#v, want one-element array", decoded["history_rows"])
	}
	history, ok := rawHistory[0].(map[string]any)
	if !ok {
		t.Fatalf("history row type = %T", rawHistory[0])
	}
	if history["detail"] != "rm file" {
		t.Fatalf("detail = %#v", history["detail"])
	}
	if decoded["footer_daemon"] != "● Daemon: OK" {
		t.Fatalf("footer_daemon = %#v", decoded["footer_daemon"])
	}
	if decoded["history_has_more"] != true {
		t.Fatalf("history_has_more = %#v", decoded["history_has_more"])
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

func TestBuildTrayContent_HistoryRows(t *testing.T) {
	t.Parallel()

	content := BuildTrayContent(PanelSnapshot{
		Items: []api.PendingApproval{{
			ID:         "pending-1",
			Client:     "cursor",
			Command:    "git status",
			AgeSeconds: 10,
		}},
		History: []api.CommandEvent{{
			ID:              "hist-1",
			ApprovalID:      "resolved-1",
			Client:          "cursor",
			FinalAction:     "allow",
			CommandRedacted: "echo done",
			CreatedAt:       "2026-07-02T10:00:00Z",
		}},
		HistoryHasMore: true,
		HealthOK:       true,
	})

	if len(content.PendingRows) != 1 {
		t.Fatalf("pending rows = %d, want 1", len(content.PendingRows))
	}
	if content.PendingRows[0].Kind != TrayRowPending {
		t.Fatalf("pending kind = %q", content.PendingRows[0].Kind)
	}
	if len(content.HistoryRows) != 1 {
		t.Fatalf("history rows = %d, want 1", len(content.HistoryRows))
	}
	if content.HistoryRows[0].Kind != TrayRowHistory {
		t.Fatalf("history kind = %q", content.HistoryRows[0].Kind)
	}
	if content.HistoryRows[0].Detail != "echo done" {
		t.Fatalf("history detail = %q", content.HistoryRows[0].Detail)
	}
	if !content.HistoryHasMore {
		t.Fatal("expected history has more")
	}
	if content.EmptyMessage != "" {
		t.Fatalf("empty message = %q, want unset when rows exist", content.EmptyMessage)
	}
}

func TestBuildTrayContent_DedupePendingApprovalInHistory(t *testing.T) {
	t.Parallel()

	content := BuildTrayContent(PanelSnapshot{
		Items: []api.PendingApproval{{
			ID:     "approval-abc",
			Client: "cursor",
		}},
		History: []api.CommandEvent{{
			ID:         "evt-dup",
			ApprovalID: "approval-abc",
			Client:     "cursor",
			FinalAction: "allow",
			CreatedAt:  "2026-07-02T10:00:00Z",
		}, {
			ID:              "evt-ok",
			ApprovalID:      "other-id",
			Client:          "claude",
			FinalAction:     "deny",
			CommandRedacted: "rm file",
			CreatedAt:       "2026-07-02T09:00:00Z",
		}},
		HealthOK: true,
	})

	if len(content.HistoryRows) != 1 {
		t.Fatalf("history rows = %d, want 1 (deduped)", len(content.HistoryRows))
	}
	if content.HistoryRows[0].ID != "evt-ok" {
		t.Fatalf("history id = %q", content.HistoryRows[0].ID)
	}
}

func TestBuildTrayContent_HistoryOnlyNoEmptyMessage(t *testing.T) {
	t.Parallel()

	content := BuildTrayContent(PanelSnapshot{
		History: []api.CommandEvent{{
			ID:              "evt-1",
			Client:          "cursor",
			FinalAction:     "allow",
			CommandRedacted: "ls",
			CreatedAt:       "2026-07-02T10:00:00Z",
		}},
		HealthOK: true,
	})

	if len(content.PendingRows) != 0 {
		t.Fatalf("pending rows = %d, want 0", len(content.PendingRows))
	}
	if len(content.HistoryRows) != 1 {
		t.Fatalf("history rows = %d, want 1", len(content.HistoryRows))
	}
	if content.EmptyMessage != "" {
		t.Fatalf("empty message = %q, want unset when history exists", content.EmptyMessage)
	}
}

func TestModeFromSegmentIndex(t *testing.T) {
	t.Parallel()

	if got := ModeFromSegmentIndex(0); got != approvalmode.Ask {
		t.Fatalf("0: %v", got)
	}
	if got := ModeFromSegmentIndex(1); got != approvalmode.Auto {
		t.Fatalf("1: %v", got)
	}
	if got := ModeFromSegmentIndex(2); got != approvalmode.AutoAllow {
		t.Fatalf("2: %v", got)
	}
	if got := ModeFromSegmentIndex(3); got != approvalmode.AutoDeny {
		t.Fatalf("3: %v", got)
	}
	if got := modeSegmentIndex(approvalmode.Auto); got != 1 {
		t.Fatalf("auto segment = %d, want 1", got)
	}
	if got := modeSegmentIndex(approvalmode.AutoDeny); got != 3 {
		t.Fatalf("auto_deny segment = %d, want 3", got)
	}
}
