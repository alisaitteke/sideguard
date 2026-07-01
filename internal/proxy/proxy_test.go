package proxy

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/store"
)

func newProxyPipes() (*io.PipeWriter, *io.PipeReader, *io.PipeReader, *io.PipeWriter) {
	proxyStdin, clientToProxy := io.Pipe()
	clientFromProxy, proxyStdout := io.Pipe()
	return clientToProxy, clientFromProxy, proxyStdin, proxyStdout
}

func TestProxyInitializePassthrough(t *testing.T) {
	serverPath := buildMinimalMCPServer(t)
	clientToProxy, clientFromProxy, proxyStdin, proxyStdout := newProxyPipes()

	done := make(chan error, 1)
	go func() {
		done <- Run(RunOptions{
			Upstream: []string{serverPath},
			Daemon:   &noopApprovalClient{},
			Stdin:    proxyStdin,
			Stdout:   proxyStdout,
		})
	}()

	initReq := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`)
	if err := WriteFrame(clientToProxy, initReq); err != nil {
		t.Fatalf("write init: %v", err)
	}

	resp, err := ReadFrame(bufio.NewReader(clientFromProxy))
	if err != nil {
		t.Fatalf("read init response: %v", err)
	}
	if !bytes.Contains(resp, []byte(`"protocolVersion"`)) {
		t.Fatalf("unexpected init response: %s", resp)
	}

	_ = clientToProxy.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("proxy exit: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("proxy did not exit after client close")
	}
}

func TestProxyToolsCallHoldAllow(t *testing.T) {
	serverPath := buildMinimalMCPServer(t)
	daemon := startTestDaemon(t)

	client := api.NewClientWithBaseURL(daemon.URL)
	clientToProxy, clientFromProxy, proxyStdin, proxyStdout := newProxyPipes()

	var proxyErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		proxyErr = Run(RunOptions{
			Upstream: []string{serverPath},
			Daemon:   client,
			Stdin:    proxyStdin,
			Stdout:   proxyStdout,
		})
	}()

	callReq := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"message":"hi"}}}`)
	if err := WriteFrame(clientToProxy, callReq); err != nil {
		t.Fatalf("write tools/call: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	pending, err := client.ListPending(context.Background())
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending count = %d, want 1", len(pending))
	}
	if pending[0].Source != "mcp" || pending[0].ToolName != "echo" {
		t.Fatalf("unexpected pending: %+v", pending[0])
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		_, _ = client.Decide(context.Background(), pending[0].ID, "allow", "")
	}()

	resp, err := ReadFrame(bufio.NewReader(clientFromProxy))
	if err != nil {
		t.Fatalf("read tools/call response: %v", err)
	}
	if !bytes.Contains(resp, []byte(`"ok"`)) {
		t.Fatalf("unexpected tools/call response: %s", resp)
	}

	_ = clientToProxy.Close()
	wg.Wait()
	if proxyErr != nil {
		t.Fatalf("proxy exit: %v", proxyErr)
	}
}

func TestProxyToolsCallDeny(t *testing.T) {
	serverPath := buildMinimalMCPServer(t)
	daemon := startTestDaemon(t)
	client := api.NewClientWithBaseURL(daemon.URL)

	clientToProxy, clientFromProxy, proxyStdin, proxyStdout := newProxyPipes()

	go func() {
		_ = Run(RunOptions{
			Upstream: []string{serverPath},
			Daemon:   client,
			Stdin:    proxyStdin,
			Stdout:   proxyStdout,
		})
	}()

	callReq := []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{}}}`)
	if err := WriteFrame(clientToProxy, callReq); err != nil {
		t.Fatalf("write tools/call: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	pending, err := client.ListPending(context.Background())
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending: %v err=%v", pending, err)
	}
	_, _ = client.Decide(context.Background(), pending[0].ID, "deny", "blocked in test")

	resp, err := ReadFrame(bufio.NewReader(clientFromProxy))
	if err != nil {
		t.Fatalf("read deny response: %v", err)
	}
	if !bytes.Contains(resp, []byte(`"error"`)) || !bytes.Contains(resp, []byte(`blocked in test`)) {
		t.Fatalf("expected JSON-RPC deny error, got: %s", resp)
	}
}

func TestProxyToolsCallPolicyAutoAllow(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "policy.yaml"), []byte(`rules:
  - match: { mcp_tool: "^echo$" }
    action: allow
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	serverPath := buildMinimalMCPServer(t)
	daemon := startTestDaemon(t)
	client := api.NewClientWithBaseURL(daemon.URL)

	clientToProxy, clientFromProxy, proxyStdin, proxyStdout := newProxyPipes()

	var proxyErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		proxyErr = Run(RunOptions{
			Upstream: []string{serverPath},
			Daemon:   client,
			Stdin:    proxyStdin,
			Stdout:   proxyStdout,
		})
	}()

	callReq := []byte(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"echo","arguments":{"message":"hi"}}}`)
	if err := WriteFrame(clientToProxy, callReq); err != nil {
		t.Fatalf("write tools/call: %v", err)
	}

	resp, err := ReadFrame(bufio.NewReader(clientFromProxy))
	if err != nil {
		t.Fatalf("read tools/call response: %v", err)
	}
	if !bytes.Contains(resp, []byte(`"ok"`)) {
		t.Fatalf("unexpected tools/call response: %s", resp)
	}

	time.Sleep(200 * time.Millisecond)
	pending, err := client.ListPending(context.Background())
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("policy allow should skip queue, got %d pending", len(pending))
	}

	_ = clientToProxy.Close()
	wg.Wait()
	if proxyErr != nil {
		t.Fatalf("proxy exit: %v", proxyErr)
	}
}

func TestProxyToolsCallPolicyAutoDeny(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".vibeguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "policy.yaml"), []byte(`rules:
  - match: { mcp_tool: ".*delete.*" }
    action: deny
    reason: "blocked by policy"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	serverPath := buildMinimalMCPServer(t)
	daemon := startTestDaemon(t)
	client := api.NewClientWithBaseURL(daemon.URL)

	clientToProxy, clientFromProxy, proxyStdin, proxyStdout := newProxyPipes()

	go func() {
		_ = Run(RunOptions{
			Upstream: []string{serverPath},
			Daemon:   client,
			Stdin:    proxyStdin,
			Stdout:   proxyStdout,
		})
	}()

	callReq := []byte(`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"delete_item","arguments":{}}}`)
	if err := WriteFrame(clientToProxy, callReq); err != nil {
		t.Fatalf("write tools/call: %v", err)
	}

	resp, err := ReadFrame(bufio.NewReader(clientFromProxy))
	if err != nil {
		t.Fatalf("read deny response: %v", err)
	}
	if !bytes.Contains(resp, []byte(`blocked by policy`)) {
		t.Fatalf("expected policy deny error, got: %s", resp)
	}

	time.Sleep(200 * time.Millisecond)
	pending, _ := client.ListPending(context.Background())
	if len(pending) != 0 {
		t.Fatalf("policy deny should not queue, got %d", len(pending))
	}
}

func TestProxyDaemonDownFailClosed(t *testing.T) {
	serverPath := buildMinimalMCPServer(t)
	clientToProxy, clientFromProxy, proxyStdin, proxyStdout := newProxyPipes()

	go func() {
		_ = Run(RunOptions{
			Upstream: []string{serverPath},
			Daemon:   api.NewClientWithBaseURL("http://127.0.0.1:1"),
			Stdin:    proxyStdin,
			Stdout:   proxyStdout,
		})
	}()

	callReq := []byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"echo","arguments":{}}}`)
	if err := WriteFrame(clientToProxy, callReq); err != nil {
		t.Fatalf("write tools/call: %v", err)
	}

	resp, err := ReadFrame(bufio.NewReader(clientFromProxy))
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if !bytes.Contains(resp, []byte(`daemon unreachable`)) {
		t.Fatalf("expected fail-closed error, got: %s", resp)
	}
}

type noopApprovalClient struct{}

func (n *noopApprovalClient) RequestApproval(context.Context, api.ApprovalRequest) (*api.ApprovalRequestResponse, error) {
	return &api.ApprovalRequestResponse{ID: "noop", Status: "pending"}, nil
}

func (n *noopApprovalClient) WaitApproval(context.Context, string) (*api.ApprovalDecisionResponse, error) {
	return &api.ApprovalDecisionResponse{Permission: "allow"}, nil
}

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

func buildMinimalMCPServer(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	srcDir := filepath.Join(filepath.Dir(file), "testdata", "minimal_mcp")
	out := filepath.Join(t.TempDir(), "minimal_mcp")
	cmd := exec.Command("go", "build", "-o", out, srcDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build minimal mcp server: %v\n%s", err, out)
	}
	return out
}
