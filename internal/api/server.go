// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Package api provides the local HTTP API for the SideGuard daemon.
// Routes bind to 127.0.0.1 only. See docs/plans/2026-07-01-0127-sideguard-foundation/
// (vgf-phase-2.0-daemon-core.md).
package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alisaitteke/sideguard/internal/store"
)

const (
	// DefaultHost is the loopback address the daemon HTTP API binds to.
	DefaultHost = "127.0.0.1"

	// DefaultPort is the default HTTP port for the daemon API.
	DefaultPort = 9477
)

// Server wraps HTTP listeners for the daemon approval API (TCP + Unix socket).
type Server struct {
	addr       string
	handler    *Handler
	http       *http.Server
	socketPath string
	lnTCP      net.Listener
	lnUnix     net.Listener
	wg         sync.WaitGroup
}

// NewServer creates an API server bound to 127.0.0.1:9477 with the given store.
func NewServer(version string, st *store.Store) *Server {
	h := &Handler{Version: version, Store: st}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", h.Health)
	mux.HandleFunc("POST /v1/approval/request", h.CreateApprovalRequest)
	mux.HandleFunc("GET /v1/approval/pending", h.ListPending)
	mux.HandleFunc("GET /v1/approval/mode", h.GetApprovalMode)
	mux.HandleFunc("PUT /v1/approval/mode", h.SetApprovalMode)
	mux.HandleFunc("GET /v1/approval/{id}/wait", h.WaitApproval)
	mux.HandleFunc("POST /v1/approval/{id}/decide", h.DecideApproval)
	mux.HandleFunc("POST /v1/events", h.IngestEvent)
	mux.HandleFunc("GET /v1/events", h.QueryEvents)
	mux.HandleFunc("POST /v1/analyze", h.AnalyzeCommand)

	addr := fmt.Sprintf("%s:%d", DefaultHost, DefaultPort)
	return &Server{
		addr:    addr,
		handler: h,
		http: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Addr returns the TCP listen address (127.0.0.1:9477).
func (s *Server) Addr() string {
	return s.addr
}

// Handler returns the root HTTP handler (shared by TCP and Unix socket).
func (s *Server) Handler() http.Handler {
	return s.http.Handler
}

// Listen starts TCP and Unix socket listeners without serving yet.
func (s *Server) Listen(socketPath string) error {
	s.socketPath = socketPath

	lnTCP, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen tcp %s: %w", s.addr, err)
	}
	s.lnTCP = lnTCP

	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		_ = lnTCP.Close()
		return fmt.Errorf("remove stale socket: %w", err)
	}

	lnUnix, err := net.Listen("unix", socketPath)
	if err != nil {
		_ = lnTCP.Close()
		return fmt.Errorf("listen unix %s: %w", socketPath, err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = lnTCP.Close()
		_ = lnUnix.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	s.lnUnix = lnUnix
	return nil
}

// Serve starts accepting connections on both listeners.
func (s *Server) Serve() error {
	if s.lnTCP == nil || s.lnUnix == nil {
		return fmt.Errorf("server not listening; call Listen first")
	}

	errCh := make(chan error, 2)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.http.Serve(s.lnTCP); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("tcp serve: %w", err)
		}
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.http.Serve(s.lnUnix); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("unix serve: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

// Shutdown gracefully stops the server and removes the Unix socket.
func (s *Server) Shutdown(ctx context.Context) error {
	err := s.http.Shutdown(ctx)
	s.wg.Wait()

	if s.socketPath != "" {
		_ = os.Remove(s.socketPath)
	}
	return err
}

// HealthURL returns the full health check URL.
func HealthURL() string {
	return fmt.Sprintf("http://%s:%d/v1/health", DefaultHost, DefaultPort)
}

// BaseURL returns the daemon HTTP API base URL.
func BaseURL() string {
	return fmt.Sprintf("http://%s:%d", DefaultHost, DefaultPort)
}

// IsLoopback reports whether addr is a loopback host.
func IsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.Trim(host, "[]")
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}
