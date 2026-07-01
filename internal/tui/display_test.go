package tui

import (
	"strings"
	"testing"

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

func TestClampSelection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		cursor, count, want int
	}{
		{0, 0, 0},
		{0, 3, 0},
		{2, 3, 2},
		{5, 3, 2},
		{-1, 3, 0},
	}
	for _, tc := range cases {
		if got := ClampSelection(tc.cursor, tc.count); got != tc.want {
			t.Fatalf("ClampSelection(%d,%d)=%d want %d", tc.cursor, tc.count, got, tc.want)
		}
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

func TestFormatAgeShortAndLong(t *testing.T) {
	t.Parallel()
	if got := FormatAgeShort(45); got != "45s" {
		t.Fatalf("short: %q", got)
	}
	if got := FormatAgeLong(45); !strings.Contains(got, "ago") {
		t.Fatalf("long: %q", got)
	}
}
