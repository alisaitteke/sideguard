package notify

import (
	"strings"
	"testing"
)

func TestNotificationsDisabledByDefault(t *testing.T) {
	t.Setenv(envNotifications, "")
	if NotificationsEnabled() {
		t.Fatal("expected notifications disabled by default")
	}
}

func TestNotificationsEnabled(t *testing.T) {
	for _, v := range []string{"1", "true", "yes", "on", "TRUE"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv(envNotifications, v)
			if !NotificationsEnabled() {
				t.Fatalf("expected enabled for %q", v)
			}
		})
	}
}

func TestPendingApprovalSkippedWhenDisabled(t *testing.T) {
	t.Setenv(envNotifications, "")
	if err := PendingApproval("id", "cursor", "cmd", "", "shell"); err != nil {
		t.Fatalf("disabled notification should be no-op, got %v", err)
	}
}

func TestFormatBody(t *testing.T) {
	body := formatBody(
		"aa4dd3d6-e91f-4946-a08e-10e8c8ef121e",
		"cursor",
		"git push origin main",
		"",
		"shell",
	)
	if !strings.Contains(body, "#aa4dd3d6") || !strings.Contains(body, "Cursor") {
		t.Fatalf("missing id/client in %q", body)
	}
	if !strings.Contains(body, "git push origin main") {
		t.Fatalf("missing command in %q", body)
	}
	if !strings.HasSuffix(body, pendingHint) {
		t.Fatalf("expected pending hint suffix, got %q", body)
	}
	if len(body) > maxBodyLen {
		t.Fatalf("body exceeds maxBodyLen: len=%d %q", len(body), body)
	}
}

func TestFormatBodyTruncatesLongCommand(t *testing.T) {
	longCmd := strings.Repeat("x", 200)
	body := formatBody("id12345678", "cursor", longCmd, "", "shell")
	if len(body) > maxBodyLen {
		t.Fatalf("body too long: len=%d %q", len(body), body)
	}
	if !strings.Contains(body, "...") {
		t.Fatalf("expected truncation ellipsis in %q", body)
	}
	if !strings.HasSuffix(body, pendingHint) {
		t.Fatalf("expected pending hint in %q", body)
	}
}

func TestFormatBodyMCP(t *testing.T) {
	body := formatBody("abcd1234-uuid", "claude", "", "filesystem_read", "mcp")
	if !strings.Contains(body, "mcp:filesystem_read") {
		t.Fatalf("expected mcp tool in body, got %q", body)
	}
}
