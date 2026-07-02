package approvalfmt

import (
	"strings"
	"testing"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
)

func TestShortApprovalID(t *testing.T) {
	t.Parallel()
	if got := ShortApprovalID("a1b2c3d4-e5f6-7890"); got != "#a1b2c3d4" {
		t.Fatalf("got %q", got)
	}
	if got := ShortApprovalID(""); got != "#?" {
		t.Fatalf("empty: got %q", got)
	}
}

func TestFormatTrayRowLabel(t *testing.T) {
	t.Parallel()
	item := api.PendingApproval{
		ID:         "a1b2c3d4-uuid",
		Client:     "cursor",
		Command:    "curl https://example.com",
		AgeSeconds: 5,
	}
	line := FormatTrayRowLabel(item)
	want := "curl https://example.com"
	if line != want {
		t.Fatalf("got %q, want %q", line, want)
	}
}

func TestFormatTrayEventLabel(t *testing.T) {
	t.Parallel()
	ev := api.CommandEvent{
		ID:              "event-uuid-1234",
		ApprovalID:      "a1b2c3d4-uuid",
		Client:          "cursor",
		FinalAction:     "allow",
		CommandRedacted: "git status",
	}
	line := FormatTrayEventLabel(ev)
	if line != "git status" {
		t.Fatalf("got %q, want git status", line)
	}
}

func TestFormatListLine(t *testing.T) {
	t.Parallel()
	item := api.PendingApproval{
		ID:         "a1b2c3d4-uuid",
		Client:     "cursor",
		Command:    "curl https://example.com",
		AgeSeconds: 5,
	}
	line := FormatListLine(item, "/Users/jo")
	want := "#a1b2c3d4 · cursor · 5s · curl https://example.com"
	if line != want {
		t.Fatalf("got %q, want %q", line, want)
	}
}

func TestFormatSummaryMCPTool(t *testing.T) {
	t.Parallel()
	item := api.PendingApproval{
		Source:   "mcp",
		ToolName: "filesystem_read",
	}
	if got := FormatSummary(item); got != "mcp:filesystem_read" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatCWD(t *testing.T) {
	t.Parallel()
	home := "/Users/jo"
	if got := FormatCWD("/Users/jo/proj", home); got != "~/proj" {
		t.Fatalf("got %q", got)
	}
	if got := FormatCWD("", home); got != "." {
		t.Fatalf("empty cwd: got %q", got)
	}
}

func TestFormatEventLine(t *testing.T) {
	t.Parallel()

	created := time.Now().UTC().Add(-30 * time.Second).Format(time.RFC3339)
	ev := api.CommandEvent{
		ID:              "event-uuid-1234",
		ApprovalID:      "a1b2c3d4-uuid",
		CreatedAt:       created,
		Client:          "cursor",
		FinalAction:     "allow",
		CommandRedacted: "git status",
	}
	line := FormatEventLine(ev, "/Users/jo")
	if !strings.Contains(line, "#a1b2c3d4") {
		t.Fatalf("expected approval id prefix, got %q", line)
	}
	if !strings.Contains(line, "cursor") {
		t.Fatalf("expected client, got %q", line)
	}
	if !strings.Contains(line, "allow") {
		t.Fatalf("expected action, got %q", line)
	}
	if !strings.Contains(line, "git status") {
		t.Fatalf("expected summary, got %q", line)
	}
}

func TestFormatEventLine_NoApprovalID(t *testing.T) {
	t.Parallel()

	ev := api.CommandEvent{
		ID:              "deadbeef-cafe",
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		Client:          "claude",
		FinalAction:     "deny",
		CommandRedacted: "rm -rf /",
	}
	line := FormatEventLine(ev, "")
	if !strings.HasPrefix(line, "#deadbeef") {
		t.Fatalf("expected event id prefix, got %q", line)
	}
}

func TestFormatEventAge(t *testing.T) {
	t.Parallel()

	created := time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339)
	if got := FormatEventAge(created); got != "2m" {
		t.Fatalf("got %q, want 2m", got)
	}
	if got := FormatEventAge("not-a-date"); got != "?" {
		t.Fatalf("invalid: got %q", got)
	}
}

func TestFormatAgeShortAndLong(t *testing.T) {
	t.Parallel()
	if got := FormatAgeShort(45); got != "45s" {
		t.Fatalf("short: %q", got)
	}
	if got := FormatAgeLong(45); !strings.Contains(got, "ago") {
		t.Fatalf("long: %q", got)
	}
}
