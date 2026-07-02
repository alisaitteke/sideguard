package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

var (
	// ErrApprovalNotFound is returned when an approval id does not exist.
	ErrApprovalNotFound = errors.New("approval not found")
)

// Client talks to the local daemon HTTP API. Connection failures are fail-closed
// for hook/MCP clients (Phase 5). See vgf-phase-2.0-daemon-core.md.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a client for the default loopback daemon endpoint.
func NewClient() *Client {
	return NewClientWithBaseURL(BaseURL())
}

// NewClientWithBaseURL creates a client for a custom daemon base URL (tests / injection).
func NewClientWithBaseURL(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Health checks daemon availability.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/health", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daemon unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	var out HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListPending returns pending approvals.
func (c *Client) ListPending(ctx context.Context) ([]PendingApproval, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/approval/pending", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daemon unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list pending failed: status %d: %s", resp.StatusCode, string(body))
	}

	var out []PendingApproval
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// Decide posts an allow/deny decision for an approval id.
func (c *Client) Decide(ctx context.Context, id, decision, reason string) (*ApprovalDecisionResponse, error) {
	body, err := json.Marshal(ApprovalDecision{Decision: decision, Reason: reason})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/approval/%s/decide", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daemon unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", ErrApprovalNotFound, id)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("decide failed: status %d: %s", resp.StatusCode, string(raw))
	}

	var out ApprovalDecisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IngestEvent posts a command event fire-and-forget. Errors are logged and swallowed
// so hook/proxy paths never block on history persistence.
func (c *Client) IngestEvent(ctx context.Context, e CommandEvent) error {
	body, err := json.Marshal(e)
	if err != nil {
		return nil
	}

	ingestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ingestCtx, http.MethodPost, c.baseURL+"/v1/events", bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	ingestClient := &http.Client{Timeout: 2 * time.Second}
	resp, err := ingestClient.Do(req)
	if err != nil {
		log.Printf("vibeguard events: ingest unreachable: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		raw, _ := io.ReadAll(resp.Body)
		log.Printf("vibeguard events: ingest failed: status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// QueryEvents lists command history rows from GET /v1/events.
func (c *Client) QueryEvents(ctx context.Context, q EventQueryParams) ([]CommandEvent, error) {
	u, err := url.Parse(c.baseURL + "/v1/events")
	if err != nil {
		return nil, err
	}
	vals := u.Query()
	if q.Since != "" {
		vals.Set("since", q.Since)
	}
	if q.Denied {
		vals.Set("denied", "true")
	}
	if q.CWD != "" {
		vals.Set("cwd", q.CWD)
	}
	if q.Limit > 0 {
		vals.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Search != "" {
		vals.Set("search", q.Search)
	}
	u.RawQuery = vals.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daemon unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query events failed: status %d: %s", resp.StatusCode, string(raw))
	}

	var out []CommandEvent
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// RequestApproval queues a new approval and returns its id.
func (c *Client) RequestApproval(ctx context.Context, req ApprovalRequest) (*ApprovalRequestResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/approval/request", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("daemon unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request approval failed: status %d: %s", resp.StatusCode, string(raw))
	}

	var out ApprovalRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// WaitApproval long-polls until the approval is decided or times out.
func (c *Client) WaitApproval(ctx context.Context, id string) (*ApprovalDecisionResponse, error) {
	waitClient := &http.Client{Timeout: 610 * time.Second}
	url := fmt.Sprintf("%s/v1/approval/%s/wait", c.baseURL, id)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := waitClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("daemon unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("wait approval failed: status %d: %s", resp.StatusCode, string(raw))
	}

	var out ApprovalDecisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetApprovalMode returns the daemon's global approval mode.
func (c *Client) GetApprovalMode(ctx context.Context) (approvalmode.Mode, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/approval/mode", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("daemon unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get approval mode failed: status %d: %s", resp.StatusCode, string(raw))
	}

	var out ApprovalModeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return approvalmode.Parse(out.Mode)
}

// SetApprovalMode updates the daemon's global approval mode.
func (c *Client) SetApprovalMode(ctx context.Context, mode approvalmode.Mode) error {
	body, err := json.Marshal(SetApprovalModeRequest{Mode: string(mode)})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/v1/approval/mode", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("daemon unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invalid approval mode: %s", string(raw))
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set approval mode failed: status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// Ping reports whether the daemon is reachable.
func Ping(ctx context.Context) error {
	_, err := NewClient().Health(ctx)
	return err
}

// IsNotFound reports whether err is an unknown approval id from Decide.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrApprovalNotFound)
}
