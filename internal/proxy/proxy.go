// Package proxy wraps upstream MCP servers over STDIO transparently.
// Dual-posture MCP proxy: client sees VibeGuard; VibeGuard forwards to upstream child.
// See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-4.0-mcp-proxy.md).
package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/alisaitteke/vibeguard/internal/api"
)

// RunOptions configures the STDIO MCP proxy loop.
type RunOptions struct {
	Upstream []string
	Daemon   ApprovalClient
	Stdin    io.Reader
	Stdout   io.Writer
}

// Wrap runs an upstream MCP server with VibeGuard interception on tools/call.
func Wrap(upstream []string) error {
	return Run(RunOptions{
		Upstream: upstream,
		Daemon:   apiDefaultClient(),
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
	})
}

func apiDefaultClient() ApprovalClient {
	return defaultAPIClient{}
}

type defaultAPIClient struct{}

func (defaultAPIClient) RequestApproval(ctx context.Context, req api.ApprovalRequest) (*api.ApprovalRequestResponse, error) {
	return api.NewClient().RequestApproval(ctx, req)
}

func (defaultAPIClient) WaitApproval(ctx context.Context, id string) (*api.ApprovalDecisionResponse, error) {
	return api.NewClient().WaitApproval(ctx, id)
}

// Run starts the transparent STDIO proxy until a pipe closes or an error occurs.
func Run(opts RunOptions) error {
	if len(opts.Upstream) == 0 {
		return fmt.Errorf("upstream command required after --")
	}
	if opts.Daemon == nil {
		opts.Daemon = apiDefaultClient()
	}
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}

	upstream, err := StartUpstream(opts.Upstream)
	if err != nil {
		return err
	}
	defer upstream.Close()

	clientIn := bufio.NewReader(opts.Stdin)
	clientOut := bufio.NewWriter(opts.Stdout)
	upstreamIn := bufio.NewWriter(upstream.Stdin)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		upstreamReader := bufio.NewReader(upstream.Stdout)
		for {
			frame, err := ReadFrame(upstreamReader)
			if err != nil {
				if err != io.EOF {
					errCh <- fmt.Errorf("upstream read: %w", err)
				}
				return
			}
			if err := WriteFrame(clientOut, frame); err != nil {
				errCh <- fmt.Errorf("client write: %w", err)
				return
			}
			if err := clientOut.Flush(); err != nil {
				errCh <- fmt.Errorf("client flush: %w", err)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() { _ = upstream.Stdin.Close() }()
		for {
			frame, err := ReadFrame(clientIn)
			if err != nil {
				if err != io.EOF {
					errCh <- fmt.Errorf("client read: %w", err)
				}
				return
			}

			action, msg, _ := ClassifyFrame(frame)
			switch action {
			case ActionHoldApproval:
				if err := handleToolsCall(opts.Daemon, msg, frame, clientOut, upstreamIn); err != nil {
					errCh <- err
					return
				}
			default:
				if err := WriteFrame(upstreamIn, frame); err != nil {
					errCh <- fmt.Errorf("upstream write: %w", err)
					return
				}
				if err := upstreamIn.Flush(); err != nil {
					errCh <- fmt.Errorf("upstream flush: %w", err)
					return
				}
			}
		}
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func handleToolsCall(client ApprovalClient, msg *JSONRPCMessage, frame []byte, clientOut *bufio.Writer, upstreamIn *bufio.Writer) error {
	params, err := ParseToolsCallParams(msg)
	if err != nil {
		if werr := writeClientError(clientOut, msg.ID, -32602, "VibeGuard: "+err.Error()); werr != nil {
			return werr
		}
		return nil
	}

	ctx := context.Background()
	if err := requestToolCallApproval(ctx, client, params); err != nil {
		if werr := writeClientError(clientOut, msg.ID, -32000, "VibeGuard: "+err.Error()); werr != nil {
			return werr
		}
		return nil
	}

	if err := WriteFrame(upstreamIn, frame); err != nil {
		return fmt.Errorf("forward tools/call: %w", err)
	}
	return upstreamIn.Flush()
}

func writeClientError(w *bufio.Writer, id json.RawMessage, code int, message string) error {
	payload, err := BuildErrorResponse(id, code, message)
	if err != nil {
		return err
	}
	if err := WriteFrame(w, payload); err != nil {
		return err
	}
	return w.Flush()
}
