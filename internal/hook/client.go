// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package hook

import (
	"context"
	"time"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
)

// DefaultApprovalTimeout matches Cursor/Claude hook timeout (600s).
const DefaultApprovalTimeout = 600 * time.Second

// DaemonClient submits approval requests and blocks until a decision.
type DaemonClient interface {
	RequestAndWait(ctx context.Context, req api.ApprovalRequest) (*api.ApprovalDecisionResponse, error)
	GetApprovalMode(ctx context.Context) (approvalmode.Mode, error)
	IngestEvent(ctx context.Context, e api.CommandEvent) error
}

// Client talks to the local SideGuard daemon with hook-sized long-poll timeouts.
// Fail-closed on connection errors — see vgf-phase-5.0-hook-bridge.md.
type Client struct {
	api     *api.Client
	timeout time.Duration
}

// NewClient creates a daemon client using the default loopback endpoint.
func NewClient() *Client {
	return NewClientWithBaseURL(api.BaseURL())
}

// NewClientWithBaseURL creates a client for tests or custom endpoints.
func NewClientWithBaseURL(baseURL string) *Client {
	return &Client{
		api:     api.NewClientWithBaseURL(baseURL),
		timeout: DefaultApprovalTimeout,
	}
}

// GetApprovalMode returns the daemon's global approval mode (defaults to Ask on error).
func (c *Client) GetApprovalMode(ctx context.Context) (approvalmode.Mode, error) {
	return c.api.GetApprovalMode(ctx)
}

// RequestAndWait queues an approval and long-polls until allow/deny or timeout.
func (c *Client) RequestAndWait(ctx context.Context, req api.ApprovalRequest) (*api.ApprovalDecisionResponse, error) {
	created, err := c.api.RequestApproval(ctx, req)
	if err != nil {
		return nil, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.api.WaitApproval(waitCtx, created.ID)
}

// IngestEvent posts a command event without blocking the hook on failures.
func (c *Client) IngestEvent(ctx context.Context, e api.CommandEvent) error {
	return c.api.IngestEvent(ctx, e)
}
