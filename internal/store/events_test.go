package store

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestIngestEventAndQuery(t *testing.T) {
	s := openTestDB(t)

	ev := CommandEvent{
		Source:          "shell",
		Client:          "cursor",
		CWD:             "/tmp",
		CommandRedacted: "git status",
		CommandNorm:     "git status",
		YamlAction:      "ask",
		DetectAction:    "allow",
		DetectRules:     []string{"safe_argv0"},
		DetectScore:     0,
		FinalAction:     "allow",
		DecisionBy:      "detect",
		Reason:          "safe command",
		LatencyMS:       12,
	}
	if err := s.IngestEvent(ev); err != nil {
		t.Fatal(err)
	}

	rows, err := s.QueryEvents(EventQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].FinalAction != "allow" || rows[0].DecisionBy != "detect" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
	if rows[0].ID == "" || rows[0].CreatedAt.IsZero() {
		t.Fatalf("expected generated id and created_at: %+v", rows[0])
	}
}

func TestIngestEventRedactsSecrets(t *testing.T) {
	s := openTestDB(t)

	secret := "curl -H 'Authorization: Bearer sk-abcdefghijklmnopqrstuvwxyz123456'"
	secretNorm := "curl -H Authorization: Bearer sk-abcdefghijklmnopqrstuvwxyz123456"
	if err := s.IngestEvent(CommandEvent{
		Source:          "shell",
		Client:          "cursor",
		CWD:             "/tmp",
		CommandRedacted: secret,
		CommandNorm:     secretNorm,
		FinalAction:     "deny",
		DecisionBy:      "detect",
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := s.QueryEvents(EventQuery{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatal("expected one row")
	}
	for _, field := range []struct {
		name, value string
	}{
		{"command_redacted", rows[0].CommandRedacted},
		{"command_norm", rows[0].CommandNorm},
	} {
		if strings.Contains(field.value, "sk-abcdefghijklmnopqrstuvwxyz123456") {
			t.Fatalf("raw secret leaked into %s: %q", field.name, field.value)
		}
		if !strings.Contains(field.value, "[REDACTED]") {
			t.Fatalf("expected redacted placeholder in %s, got %q", field.name, field.value)
		}
	}
}

func TestIngestEventTruncatesLongCommand(t *testing.T) {
	s := openTestDB(t)

	long := strings.Repeat("a", 20<<10)
	if err := s.IngestEvent(CommandEvent{
		Source:          "shell",
		Client:          "cursor",
		CWD:             "/tmp",
		CommandRedacted: long,
		CommandNorm:     long,
		FinalAction:     "allow",
		DecisionBy:      "yaml",
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := s.QueryEvents(EventQuery{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows[0].CommandNorm) > maxCommandNormLen {
		t.Fatalf("command_norm len = %d, want <= %d", len(rows[0].CommandNorm), maxCommandNormLen)
	}
	if len(rows[0].CommandRedacted) > maxCommandRedactedLen {
		t.Fatalf("command_redacted len = %d, want <= %d", len(rows[0].CommandRedacted), maxCommandRedactedLen)
	}
}

func TestQueryEventsDeniedFilter(t *testing.T) {
	s := openTestDB(t)

	for _, action := range []string{"allow", "deny", "deny"} {
		if err := s.IngestEvent(CommandEvent{
			Source:      "shell",
			Client:      "cursor",
			CWD:         "/tmp",
			FinalAction: action,
			DecisionBy:  "detect",
		}); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := s.QueryEvents(EventQuery{Denied: true, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("denied rows = %d, want 2", len(rows))
	}
	for _, row := range rows {
		if row.FinalAction != "deny" {
			t.Fatalf("expected deny, got %q", row.FinalAction)
		}
	}
}

func TestQueryEventsSinceAndSearch(t *testing.T) {
	s := openTestDB(t)
	past := time.Now().UTC().Add(-2 * time.Hour)
	if err := s.IngestEvent(CommandEvent{
		ID:              "old-event",
		CreatedAt:       past,
		Source:          "shell",
		Client:          "cursor",
		CWD:             "/tmp",
		CommandRedacted: "old git status",
		CommandNorm:     "git status",
		FinalAction:     "allow",
		DecisionBy:      "detect",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.IngestEvent(CommandEvent{
		Source:          "shell",
		Client:          "cursor",
		CWD:             "/tmp",
		CommandRedacted: "npm install lodash",
		CommandNorm:     "npm install",
		FinalAction:     "allow",
		DecisionBy:      "detect",
	}); err != nil {
		t.Fatal(err)
	}

	since := time.Now().UTC().Add(-1 * time.Hour)
	rows, err := s.QueryEvents(EventQuery{Since: since, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("since rows = %d, want 1", len(rows))
	}

	searchRows, err := s.QueryEvents(EventQuery{Search: "npm", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(searchRows) != 1 || !strings.Contains(searchRows[0].CommandNorm, "npm") {
		t.Fatalf("search rows = %+v", searchRows)
	}
}

func TestMigrateV3OnExistingDB(t *testing.T) {
	s := openTestDB(t)
	if _, err := s.db.Exec(`INSERT INTO approvals (id, source, client, command, cwd, status, created_at)
		VALUES ('legacy', 'shell', 'cursor', 'ls', '/tmp', 'pending', ?)`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}

	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}

	if err := s.IngestEvent(CommandEvent{
		Source:      "shell",
		Client:      "cursor",
		CWD:         "/tmp",
		FinalAction: "allow",
		DecisionBy:  "yaml",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPruneEventsByAgeAndCount(t *testing.T) {
	s := openTestDB(t)
	ns := "e2e-sdh-prune"
	t.Cleanup(func() {
		_, _ = s.db.Exec(`DELETE FROM command_events WHERE reason LIKE ?`, ns+"%")
	})

	oldTime := time.Now().UTC().Add(-48 * time.Hour)
	for i := 0; i < 5; i++ {
		if err := s.IngestEvent(CommandEvent{
			ID:              fmt.Sprintf("%s-old-%d", ns, i),
			CreatedAt:       oldTime,
			Source:          "shell",
			Client:          "cursor",
			CWD:             "/tmp",
			CommandRedacted: "old command",
			FinalAction:     "allow",
			DecisionBy:      "detect",
			Reason:          ns + "-old",
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := s.IngestEvent(CommandEvent{
			ID:          fmt.Sprintf("%s-new-%d", ns, i),
			Source:      "shell",
			Client:      "cursor",
			CWD:         "/tmp",
			FinalAction: "allow",
			DecisionBy:  "detect",
			Reason:      ns + "-new",
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.PruneEvents(1, 0); err != nil {
		t.Fatal(err)
	}
	rows, err := s.QueryEvents(EventQuery{Search: ns, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if strings.Contains(row.Reason, "-old") {
			t.Fatalf("old row survived age prune: %+v", row)
		}
	}

	for i := 0; i < 10; i++ {
		if err := s.IngestEvent(CommandEvent{
			ID:          fmt.Sprintf("%s-fill-%d", ns, i),
			Source:      "shell",
			Client:      "cursor",
			CWD:         "/tmp",
			FinalAction: "allow",
			DecisionBy:  "detect",
			Reason:      ns + "-fill",
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.PruneEvents(0, 5); err != nil {
		t.Fatal(err)
	}
	rows, err = s.QueryEvents(EventQuery{Search: ns, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) > 5 {
		t.Fatalf("count prune left %d rows, want <= 5", len(rows))
	}
}

func TestIngestEventToolNameOnly(t *testing.T) {
	s := openTestDB(t)

	if err := s.IngestEvent(CommandEvent{
		Source:          "mcp",
		Client:          "cursor",
		CWD:             "/tmp",
		CommandRedacted: "",
		CommandNorm:     "mcp:read_file",
		ToolName:        "read_file",
		FinalAction:     "allow",
		DecisionBy:      "yaml",
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := s.QueryEvents(EventQuery{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].ToolName != "read_file" {
		t.Fatalf("tool_name = %q, want read_file", rows[0].ToolName)
	}
}
