// HTTP Stream (MCP 2025-11) reverse proxy for remote MCP servers.
// Binds localhost only; validates Origin; intercepts tools/call for approval.
// See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-8.0-hardening.md).
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPListen = "127.0.0.1:0"
	maxHTTPBodyBytes  = 8 << 20 // 8 MiB
)

// HTTPOptions configures the Streamable HTTP MCP reverse proxy.
type HTTPOptions struct {
	ListenAddr  string
	UpstreamURL string
	Client      ApprovalClient
}

// RunHTTP serves a localhost Streamable HTTP reverse proxy until ctx is done.
func RunHTTP(ctx context.Context, opts HTTPOptions) error {
	if strings.TrimSpace(opts.UpstreamURL) == "" {
		return fmt.Errorf("upstream URL required")
	}
	upstream, err := url.Parse(opts.UpstreamURL)
	if err != nil {
		return fmt.Errorf("parse upstream URL: %w", err)
	}
	if opts.Client == nil {
		opts.Client = apiDefaultClient()
	}

	listen := strings.TrimSpace(opts.ListenAddr)
	if listen == "" {
		listen = defaultHTTPListen
	}

	handler := &httpStreamProxy{
		upstream: upstream,
		client:   opts.Client,
		transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: 0,
		},
	}

	srv := &http.Server{
		Addr:              listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("listen %s: %w", listen, err)
	}
	log.Printf("vibeguard: HTTP MCP proxy listening on http://%s -> %s", ln.Addr().String(), upstream.String())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

type httpStreamProxy struct {
	upstream  *url.URL
	client    ApprovalClient
	transport *http.Transport
}

func (p *httpStreamProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !IsAllowedOrigin(r.Header.Get("Origin")) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodPost:
		p.handlePOST(w, r)
	case http.MethodGet:
		p.reverseProxy(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (p *httpStreamProxy) handlePOST(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxHTTPBodyBytes))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	if len(body) > 0 {
		action, msg, _ := ClassifyFrame(body)
		if action == ActionHoldApproval {
			params, err := ParseToolsCallParams(msg)
			if err != nil {
				p.writeJSONRPCError(w, msg.ID, -32602, "VibeGuard: "+err.Error())
				return
			}
			if err := requestToolCallApproval(r.Context(), p.client, params); err != nil {
				p.writeJSONRPCError(w, msg.ID, -32000, "VibeGuard: "+err.Error())
				return
			}
		}
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	p.reverseProxy(w, r)
}

func (p *httpStreamProxy) reverseProxy(w http.ResponseWriter, r *http.Request) {
	proxy := httputil.NewSingleHostReverseProxy(p.upstream)
	proxy.Transport = p.transport
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = p.upstream.Scheme
		req.URL.Host = p.upstream.Host
		req.URL.Path = singleJoinPath(p.upstream.Path, req.URL.Path)
		req.Host = p.upstream.Host
		if req.URL.RawQuery == "" {
			req.URL.RawQuery = p.upstream.RawQuery
		}
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Hint reverse proxies not to buffer SSE streams (MCP Streamable HTTP).
		if resp.Header.Get("Content-Type") == "text/event-stream" {
			resp.Header.Set("X-Accel-Buffering", "no")
		}
		return nil
	}
	proxy.ServeHTTP(w, r)
}

func (p *httpStreamProxy) writeJSONRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	payload, err := BuildErrorResponse(id, code, message)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

// IsAllowedOrigin validates the Origin header for DNS rebinding protection.
// Empty origin is allowed (non-browser MCP clients). Browser origins must be localhost.
func IsAllowedOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return u.Scheme == "http" || u.Scheme == "https"
	default:
		return false
	}
}

func singleJoinPath(basePath, reqPath string) string {
	basePath = strings.TrimSuffix(basePath, "/")
	if basePath == "" {
		return reqPath
	}
	if reqPath == "" || reqPath == "/" {
		return basePath
	}
	return basePath + reqPath
}
