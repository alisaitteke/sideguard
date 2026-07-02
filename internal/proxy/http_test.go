package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

func TestIsAllowedOrigin(t *testing.T) {
	tests := []struct {
		origin string
		want   bool
	}{
		{"", true},
		{"http://localhost:3000", true},
		{"http://127.0.0.1:8080", true},
		{"http://[::1]:5173", true},
		{"https://localhost", true},
		{"http://evil.example.com", false},
		{"ftp://localhost", false},
	}
	for _, tc := range tests {
		if got := IsAllowedOrigin(tc.origin); got != tc.want {
			t.Fatalf("IsAllowedOrigin(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
}

type stubApprovalClient struct {
	allow bool
}

func (s stubApprovalClient) RequestApproval(_ context.Context, _ api.ApprovalRequest) (*api.ApprovalRequestResponse, error) {
	return &api.ApprovalRequestResponse{ID: "test-id"}, nil
}

func (s stubApprovalClient) WaitApproval(_ context.Context, _ string) (*api.ApprovalDecisionResponse, error) {
	perm := "deny"
	if s.allow {
		perm = "allow"
	}
	return &api.ApprovalDecisionResponse{Permission: perm}, nil
}

func (s stubApprovalClient) GetApprovalMode(context.Context) (approvalmode.Mode, error) {
	return approvalmode.Ask, nil
}

func (s stubApprovalClient) IngestEvent(context.Context, api.CommandEvent) error {
	return nil
}

func TestHTTPProxyToolsCallDenied(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream should not be called when approval denied")
	}))
	t.Cleanup(upstream.Close)

	proxyURL := startHTTPProxyTest(t, upstream.URL, stubApprovalClient{allow: false})

	frame, _ := json.Marshal(JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"delete_file","arguments":{}}`),
	})

	resp, err := http.Post(proxyURL, "application/json", bytes.NewReader(frame))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var msg JSONRPCMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("decode response: %v raw=%s", err, body)
	}
	if msg.Error == nil {
		t.Fatalf("expected JSON-RPC error, got %+v", msg)
	}
}

func TestHTTPProxyToolsCallAllowed(t *testing.T) {
	var upstreamCalled bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
	}))
	t.Cleanup(upstream.Close)

	proxyURL := startHTTPProxyTest(t, upstream.URL, stubApprovalClient{allow: true})

	frame, _ := json.Marshal(JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"read_file","arguments":{}}`),
	})

	resp, err := http.Post(proxyURL, "application/json", bytes.NewReader(frame))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !upstreamCalled {
		t.Fatal("expected upstream to receive forwarded tools/call")
	}
}

func TestHTTPProxyRejectsBadOrigin(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	proxyURL := startHTTPProxyTest(t, upstream.URL, stubApprovalClient{allow: true})

	req, err := http.NewRequest(http.MethodPost, proxyURL, bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", "http://attacker.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func startHTTPProxyTest(t *testing.T, upstreamURL string, client ApprovalClient) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = RunHTTP(ctx, HTTPOptions{
			ListenAddr:  addr,
			UpstreamURL: upstreamURL,
			Client:      client,
		})
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return "http://" + addr
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("proxy did not become ready")
	return ""
}
