// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package proxy

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
	"github.com/alisaitteke/sideguard/internal/llm"
	"github.com/alisaitteke/sideguard/internal/policy"

	_ "github.com/alisaitteke/sideguard/internal/detect"
)

// ApprovalClient submits and waits on daemon approval requests.
type ApprovalClient interface {
	RequestApproval(ctx context.Context, req api.ApprovalRequest) (*api.ApprovalRequestResponse, error)
	WaitApproval(ctx context.Context, id string) (*api.ApprovalDecisionResponse, error)
	GetApprovalMode(ctx context.Context) (approvalmode.Mode, error)
	IngestEvent(ctx context.Context, e api.CommandEvent) error
}

// detectClient infers the host IDE from environment variables.
func detectClient() string {
	if os.Getenv("CURSOR_TRACE_ID") != "" || os.Getenv("CURSOR_SESSION_ID") != "" {
		return "cursor"
	}
	if os.Getenv("CLAUDE_CODE") != "" || os.Getenv("CLAUDECODE") != "" {
		return "claude"
	}
	return "unknown"
}

// requestToolCallApproval blocks until the user allows or denies a tools/call.
// Fail-closed: daemon errors result in denial without forwarding to upstream.
func requestToolCallApproval(ctx context.Context, client ApprovalClient, params ToolsCallParams) error {
	start := time.Now()
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	command := fmt.Sprintf("mcp:%s", params.Name)
	req := api.ApprovalRequest{
		Source:    "mcp",
		Client:    detectClient(),
		Command:   command,
		CWD:       cwd,
		ToolName:  params.Name,
		ToolInput: params.Arguments,
	}
	fireIngest := func(ev api.CommandEvent) {
		go client.IngestEvent(ctx, ev)
	}

	policyInput := policy.Input{
		Command:  command,
		ToolName: params.Name,
		CWD:      cwd,
	}
	llmEnabled := llm.Enabled(cwd)
	clf, clfErr := llm.ClassifierFor(cwd)
	if clfErr != nil {
		log.Printf("sideguard llm: classifier init failed (fail-safe ask): %v", clfErr)
	}
	mode := approvalmode.Ask
	if m, err := client.GetApprovalMode(ctx); err == nil {
		mode = m
	}
	policyResult := policy.EvaluateFull(ctx, cwd, policyInput, policy.EvaluateOpts{
		LLMEnabled: llmEnabled,
		Classifier: clf,
		Mode:       mode,
	})
	evalLatency := time.Since(start)
	switch policyResult.Action {
	case policy.ActionAllow:
		fireIngest(api.BuildCommandEvent(req, policyResult, "allow", api.InferDecisionBy(policyResult), policyResult.Reason, "", evalLatency))
		return nil
	case policy.ActionDeny:
		msg := policyResult.Reason
		if msg == "" {
			msg = "tool call blocked by policy"
		}
		fireIngest(api.BuildCommandEvent(req, policyResult, "deny", api.InferDecisionBy(policyResult), msg, "", evalLatency))
		return fmt.Errorf("%s", msg)
	}

	created, err := client.RequestApproval(ctx, req)
	if err != nil {
		fireIngest(api.BuildCommandEvent(req, policyResult, "deny", "mode", "daemon unreachable", "", time.Since(start)))
		return fmt.Errorf("daemon unreachable: %w", err)
	}

	decision, err := client.WaitApproval(ctx, created.ID)
	totalLatency := time.Since(start)
	if err != nil {
		fireIngest(api.BuildCommandEvent(req, policyResult, "deny", "mode", "approval wait failed", created.ID, totalLatency))
		return fmt.Errorf("approval wait failed: %w", err)
	}
	decisionBy := api.QueueDecisionBy(mode)
	if decision.Permission != "allow" {
		msg := decision.UserMessage
		if msg == "" {
			msg = "tool call denied by user"
		}
		fireIngest(api.BuildCommandEvent(req, policyResult, "deny", decisionBy, msg, created.ID, totalLatency))
		return fmt.Errorf("%s", msg)
	}
	fireIngest(api.BuildCommandEvent(req, policyResult, "allow", decisionBy, decision.UserMessage, created.ID, totalLatency))
	return nil
}
