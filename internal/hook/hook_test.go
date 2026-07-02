package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/policy"
	"github.com/alisaitteke/vibeguard/internal/store"

	_ "github.com/alisaitteke/vibeguard/internal/detect"
)

func startTestDaemon(t *testing.T) *httptest.Server {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := api.NewServer("test", st)
	return httptest.NewServer(srv.Handler())
}

func waitHook(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("timed out waiting for hook to finish")
	}
}

// queueTriggerCmd is a non-safe argv0 that detect returns ask for, exercising the daemon queue.
const queueTriggerCmd = "custom_unknown_cmd arg"

func TestRunShellCursorAllow(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{"command":"` + queueTriggerCmd + `","cwd":"/tmp"}`
	stdin := strings.NewReader(input)
	var stdout bytes.Buffer

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		code := RunShell(stdin, &stdout, client)
		if code != ExitAllow {
			t.Errorf("exit code = %d, want %d", code, ExitAllow)
		}
	}()

	time.Sleep(150 * time.Millisecond)
	apiClient := api.NewClientWithBaseURL(daemon.URL)
	pending, err := apiClient.ListPending(context.Background())
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending: %+v err=%v", pending, err)
	}
	if pending[0].Source != "shell" || pending[0].Command != queueTriggerCmd {
		t.Fatalf("unexpected pending: %+v", pending[0])
	}

	_, _ = apiClient.Decide(context.Background(), pending[0].ID, "allow", "")
	waitHook(t, &wg)

	var resp CursorPermissionResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("decode stdout: %v raw=%q", err, stdout.String())
	}
	if resp.Permission != "allow" {
		t.Fatalf("permission = %q, want allow", resp.Permission)
	}
}

func TestRunShellCursorWithHookEventName(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{
		"hook_event_name": "beforeShellExecution",
		"command": "` + queueTriggerCmd + `",
		"cwd": "/tmp",
		"sandbox": false,
		"conversation_id": "test-conv",
		"workspace_roots": ["/tmp"]
	}`
	var stdout bytes.Buffer

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		code := RunShell(strings.NewReader(input), &stdout, client)
		if code != ExitAllow {
			t.Errorf("exit code = %d, want %d", code, ExitAllow)
		}
	}()

	time.Sleep(150 * time.Millisecond)
	apiClient := api.NewClientWithBaseURL(daemon.URL)
	pending, err := apiClient.ListPending(context.Background())
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending: %+v err=%v", pending, err)
	}
	if pending[0].Command != queueTriggerCmd {
		t.Fatalf("unexpected pending command: %+v", pending[0])
	}
	_, _ = apiClient.Decide(context.Background(), pending[0].ID, "allow", "")
	waitHook(t, &wg)

	var resp CursorPermissionResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("decode stdout: %v raw=%q", err, stdout.String())
	}
	if resp.Permission != "allow" {
		t.Fatalf("permission = %q, want allow", resp.Permission)
	}
	if strings.Contains(stdout.String(), "hookSpecificOutput") {
		t.Fatalf("expected Cursor response shape, got %q", stdout.String())
	}
}

func TestRunShellCursorToolInputCommand(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "preToolUse Shell",
			input: `{
				"hook_event_name": "preToolUse",
				"tool_name": "Shell",
				"tool_input": {"command": "` + queueTriggerCmd + `", "working_directory": "/tmp"},
				"cwd": "/tmp"
			}`,
			want: queueTriggerCmd,
		},
		{
			name: "beforeShellExecution nested command",
			input: `{
				"hook_event_name": "beforeShellExecution",
				"tool_name": "Shell",
				"tool_input": {"command": "` + queueTriggerCmd + `"},
				"cwd": "/tmp"
			}`,
			want: queueTriggerCmd,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				code := RunShell(strings.NewReader(tc.input), &stdout, client)
				if code != ExitAllow {
					t.Errorf("exit code = %d, want %d stdout=%q", code, ExitAllow, stdout.String())
				}
			}()

			time.Sleep(150 * time.Millisecond)
			apiClient := api.NewClientWithBaseURL(daemon.URL)
			pending, err := apiClient.ListPending(context.Background())
			if err != nil || len(pending) != 1 {
				t.Fatalf("pending: %+v err=%v", pending, err)
			}
			if pending[0].Command != tc.want {
				t.Fatalf("unexpected pending command: %+v", pending[0])
			}
			_, _ = apiClient.Decide(context.Background(), pending[0].ID, "allow", "")
			waitHook(t, &wg)

			var resp CursorPermissionResponse
			if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
				t.Fatalf("decode stdout: %v raw=%q", err, stdout.String())
			}
			if resp.Permission != "allow" {
				t.Fatalf("permission = %q, want allow", resp.Permission)
			}
		})
	}
}

func TestRunShellCursorDeny(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{"command":"` + queueTriggerCmd + `","cwd":"."}`
	stdin := strings.NewReader(input)
	var stdout bytes.Buffer

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		code := RunShell(stdin, &stdout, client)
		if code != ExitDeny {
			t.Errorf("exit code = %d, want %d", code, ExitDeny)
		}
	}()

	time.Sleep(150 * time.Millisecond)
	apiClient := api.NewClientWithBaseURL(daemon.URL)
	pending, _ := apiClient.ListPending(context.Background())
	_, _ = apiClient.Decide(context.Background(), pending[0].ID, "deny", "blocked external curl")
	waitHook(t, &wg)

	var resp CursorPermissionResponse
	_ = json.Unmarshal(stdout.Bytes(), &resp)
	if resp.Permission != "deny" {
		t.Fatalf("permission = %q, want deny", resp.Permission)
	}
}

func TestRunShellDaemonUnreachable(t *testing.T) {
	client := NewClientWithBaseURL("http://127.0.0.1:1")
	input := `{"command":"` + queueTriggerCmd + `","cwd":"."}`
	var stdout bytes.Buffer

	code := RunShell(strings.NewReader(input), &stdout, client)
	if code != ExitDeny {
		t.Fatalf("exit code = %d, want %d", code, ExitDeny)
	}

	var resp CursorPermissionResponse
	_ = json.Unmarshal(stdout.Bytes(), &resp)
	if resp.Permission != "deny" || !strings.Contains(resp.UserMessage, "daemon") {
		t.Fatalf("expected fail-closed deny, got %+v", resp)
	}
}

func TestRunShellClaudeAllow(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "npm test"},
		"cwd": "/project"
	}`
	var stdout bytes.Buffer

	code := RunShell(strings.NewReader(input), &stdout, client)
	if code != ExitAllow {
		t.Fatalf("exit code = %d, want allow for safe npm test", code)
	}

	pending, _ := api.NewClientWithBaseURL(daemon.URL).ListPending(context.Background())
	if len(pending) != 0 {
		t.Fatalf("safe npm test should auto-allow via detect, got %d pending", len(pending))
	}

	var resp ClaudeHookResponse
	_ = json.Unmarshal(stdout.Bytes(), &resp)
	if resp.HookSpecificOutput.PermissionDecision != "allow" {
		t.Fatalf("decision = %q, want allow", resp.HookSpecificOutput.PermissionDecision)
	}
	if resp.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Fatalf("hookEventName = %q", resp.HookSpecificOutput.HookEventName)
	}
}

func TestRunShellClaudeNonBashPassThrough(t *testing.T) {
	client := NewClientWithBaseURL("http://127.0.0.1:1")

	input := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Read",
		"tool_input": {"path": "/tmp/foo"},
		"cwd": "/project"
	}`
	var stdout bytes.Buffer

	code := RunShell(strings.NewReader(input), &stdout, client)
	if code != ExitAllow {
		t.Fatalf("exit code = %d, want allow passthrough for non-shell tool", code)
	}

	var resp ClaudeHookResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("decode stdout: %v raw=%q", err, stdout.String())
	}
	if resp.HookSpecificOutput.PermissionDecision != "allow" {
		t.Fatalf("decision = %q, want allow", resp.HookSpecificOutput.PermissionDecision)
	}
}

func TestRunMCPCursorWithHookEventName(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{
		"hook_event_name": "beforeMCPExecution",
		"tool_name": "filesystem_read",
		"tool_input": {"path": "/tmp/foo"},
		"cwd": "/tmp"
	}`
	var stdout bytes.Buffer

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		code := RunMCP(strings.NewReader(input), &stdout, client)
		if code != ExitAllow {
			t.Errorf("exit code = %d, want %d", code, ExitAllow)
		}
	}()

	time.Sleep(150 * time.Millisecond)
	apiClient := api.NewClientWithBaseURL(daemon.URL)
	pending, _ := apiClient.ListPending(context.Background())
	if len(pending) != 1 || pending[0].ToolName != "filesystem_read" {
		t.Fatalf("unexpected pending: %+v", pending)
	}
	_, _ = apiClient.Decide(context.Background(), pending[0].ID, "allow", "")
	waitHook(t, &wg)

	var resp CursorPermissionResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("decode stdout: %v raw=%q", err, stdout.String())
	}
	if resp.Permission != "allow" {
		t.Fatalf("permission = %q, want allow", resp.Permission)
	}
}

func TestRunMCPCursorAllow(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{"tool_name":"filesystem_read","tool_input":{"path":"/tmp/foo"}}`
	var stdout bytes.Buffer

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		code := RunMCP(strings.NewReader(input), &stdout, client)
		if code != ExitAllow {
			t.Errorf("exit code = %d, want %d", code, ExitAllow)
		}
	}()

	time.Sleep(150 * time.Millisecond)
	apiClient := api.NewClientWithBaseURL(daemon.URL)
	pending, _ := apiClient.ListPending(context.Background())
	if pending[0].Source != "mcp" || pending[0].ToolName != "filesystem_read" {
		t.Fatalf("unexpected pending: %+v", pending[0])
	}
	_, _ = apiClient.Decide(context.Background(), pending[0].ID, "allow", "")
	waitHook(t, &wg)

	var resp CursorPermissionResponse
	_ = json.Unmarshal(stdout.Bytes(), &resp)
	if resp.Permission != "allow" {
		t.Fatalf("permission = %q", resp.Permission)
	}
}

func TestRunMCPClaudeDeny(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "mcp__filesystem__read",
		"tool_input": {"path": "/tmp/x"},
		"cwd": "."
	}`
	var stdout bytes.Buffer

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		code := RunMCP(strings.NewReader(input), &stdout, client)
		if code != ExitDeny {
			t.Errorf("exit code = %d, want %d", code, ExitDeny)
		}
	}()

	time.Sleep(150 * time.Millisecond)
	apiClient := api.NewClientWithBaseURL(daemon.URL)
	pending, _ := apiClient.ListPending(context.Background())
	_, _ = apiClient.Decide(context.Background(), pending[0].ID, "deny", "destructive mcp call")
	waitHook(t, &wg)

	var resp ClaudeHookResponse
	_ = json.Unmarshal(stdout.Bytes(), &resp)
	if resp.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("decision = %q, want deny", resp.HookSpecificOutput.PermissionDecision)
	}
}

func TestDecodeToolInputStringJSON(t *testing.T) {
	raw := json.RawMessage(`"{\"path\":\"/tmp\"}"`)
	out, err := decodeToolInput(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["path"] != "/tmp" {
		t.Fatalf("unexpected: %+v", out)
	}
}

func writeTestPolicy(t *testing.T, content string) {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "policy.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
}

func TestRunShellDetectAutoAllow(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{"command":"git status","cwd":"/tmp"}`
	var stdout bytes.Buffer
	code := RunShell(strings.NewReader(input), &stdout, client)
	if code != ExitAllow {
		t.Fatalf("exit code = %d, want allow for git status", code)
	}

	pending, _ := api.NewClientWithBaseURL(daemon.URL).ListPending(context.Background())
	if len(pending) != 0 {
		t.Fatalf("git status should auto-allow via detect, got %d pending", len(pending))
	}
}

func TestRunShellDetectAutoDeny(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{"command":"rm -rf /","cwd":"/tmp"}`
	var stdout bytes.Buffer
	code := RunShell(strings.NewReader(input), &stdout, client)
	if code != ExitDeny {
		t.Fatalf("exit code = %d, want deny for rm -rf /", code)
	}

	pending, _ := api.NewClientWithBaseURL(daemon.URL).ListPending(context.Background())
	if len(pending) != 0 {
		t.Fatalf("destructive command should deny at detect, got %d pending", len(pending))
	}
}

// TestRunShellIngestsEventBeforeReturn guards the Phase 10 fix: RunShell must
// not return until the ingest POST completed, because the hook binary exits
// immediately afterwards (os.Exit) and would otherwise drop the event.
func TestRunShellIngestsEventBeforeReturn(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{"command":"git status","cwd":"/tmp"}`
	var stdout bytes.Buffer
	if code := RunShell(strings.NewReader(input), &stdout, client); code != ExitAllow {
		t.Fatalf("exit code = %d, want allow", code)
	}

	// The daemon inserts asynchronously after responding 202 — poll briefly.
	apiClient := api.NewClientWithBaseURL(daemon.URL)
	deadline := time.Now().Add(2 * time.Second)
	for {
		events, err := apiClient.QueryEvents(context.Background(), api.EventQueryParams{Limit: 10})
		if err == nil && len(events) == 1 {
			if events[0].CommandRedacted != "git status" || events[0].FinalAction != "allow" {
				t.Fatalf("unexpected event: %+v", events[0])
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("event not ingested before deadline: events=%+v err=%v", events, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestRunShellControlPlaneAutoAllow(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	for _, cmd := range []string{
		`{"command":"vibeguard pending","cwd":"/tmp"}`,
		`{"command":"vibeguard ui","cwd":"/tmp"}`,
		`{"command":"vibeguard approve abc-123","cwd":"/tmp"}`,
		`{"command":"vibeguard daemon status","cwd":"/tmp"}`,
	} {
		var stdout bytes.Buffer
		code := RunShell(strings.NewReader(cmd), &stdout, client)
		if code != ExitAllow {
			t.Fatalf("cmd %s: exit code = %d, want allow", cmd, code)
		}
	}

	pending, _ := api.NewClientWithBaseURL(daemon.URL).ListPending(context.Background())
	if len(pending) != 0 {
		t.Fatalf("control-plane commands should not queue approvals, got %d pending", len(pending))
	}
}

func TestRunShellDevBypass(t *testing.T) {
	t.Setenv("VIBEGUARD_DEV", "1")
	t.Cleanup(func() { t.Setenv("VIBEGUARD_DEV", "") })

	// Unreachable daemon — bypass should still allow without contacting it.
	client := NewClientWithBaseURL("http://127.0.0.1:1")
	input := `{"command":"curl example.com","cwd":"/tmp"}`
	var stdout bytes.Buffer
	code := RunShell(strings.NewReader(input), &stdout, client)
	if code != ExitAllow {
		t.Fatalf("exit code = %d, want allow with VIBEGUARD_DEV=1", code)
	}
}

func TestRunShellPolicyAutoAllow(t *testing.T) {
	writeTestPolicy(t, `rules:
  - match: { command: "^git status" }
    action: allow
`)
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{"command":"git status","cwd":"/tmp"}`
	var stdout bytes.Buffer
	code := RunShell(strings.NewReader(input), &stdout, client)
	if code != ExitAllow {
		t.Fatalf("exit code = %d, want allow", code)
	}

	pending, _ := api.NewClientWithBaseURL(daemon.URL).ListPending(context.Background())
	if len(pending) != 0 {
		t.Fatalf("expected no pending approvals, got %d", len(pending))
	}
}

func TestRunShellWorkspaceDevPolicyAutoAllow(t *testing.T) {
	repo := t.TempDir()
	content, err := policy.DevWorkspacePolicy(repo)
	if err != nil {
		t.Fatal(err)
	}
	wsDir := filepath.Join(repo, ".vibeguard")
	if err := os.MkdirAll(wsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "policy.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := fmt.Sprintf(`{"command":"go test ./internal/policy/...","cwd":%q}`, repo)
	var stdout bytes.Buffer
	code := RunShell(strings.NewReader(input), &stdout, client)
	if code != ExitAllow {
		t.Fatalf("exit code = %d, want allow for in-repo go test", code)
	}

	pending, _ := api.NewClientWithBaseURL(daemon.URL).ListPending(context.Background())
	if len(pending) != 0 {
		t.Fatalf("workspace dev policy should auto-allow, got %d pending", len(pending))
	}
}

func TestRunShellMalformedInput(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	var stdout bytes.Buffer
	code := RunShell(strings.NewReader(`not-json`), &stdout, client)
	if code != ExitDeny {
		t.Fatalf("exit code = %d, want deny on malformed input", code)
	}

	pending, _ := api.NewClientWithBaseURL(daemon.URL).ListPending(context.Background())
	if len(pending) != 0 {
		t.Fatalf("expected no pending on malformed input, got %d", len(pending))
	}
}

func TestRunMCPMalformedInput(t *testing.T) {
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	var stdout bytes.Buffer
	code := RunMCP(strings.NewReader(`{`), &stdout, client)
	if code != ExitDeny {
		t.Fatalf("exit code = %d, want deny on malformed input", code)
	}
}

func TestRunMCPPolicyAutoDeny(t *testing.T) {
	writeTestPolicy(t, `rules:
  - match: { mcp_tool: ".*delete.*" }
    action: deny
    reason: "destructive tool blocked"
`)
	daemon := startTestDaemon(t)
	client := NewClientWithBaseURL(daemon.URL)

	input := `{"tool_name":"memory_delete","tool_input":{"id":"x"}}`
	var stdout bytes.Buffer
	code := RunMCP(strings.NewReader(input), &stdout, client)
	if code != ExitDeny {
		t.Fatalf("exit code = %d, want deny", code)
	}

	pending, _ := api.NewClientWithBaseURL(daemon.URL).ListPending(context.Background())
	if len(pending) != 0 {
		t.Fatalf("expected no pending approvals on policy deny, got %d", len(pending))
	}

	var resp CursorPermissionResponse
	_ = json.Unmarshal(stdout.Bytes(), &resp)
	if !strings.Contains(resp.UserMessage, "destructive") {
		t.Fatalf("expected policy reason, got %+v", resp)
	}
}
