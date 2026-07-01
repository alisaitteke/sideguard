package proxy

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/llm"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

// ApprovalClient submits and waits on daemon approval requests.
type ApprovalClient interface {
	RequestApproval(ctx context.Context, req api.ApprovalRequest) (*api.ApprovalRequestResponse, error)
	WaitApproval(ctx context.Context, id string) (*api.ApprovalDecisionResponse, error)
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
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	command := fmt.Sprintf("mcp:%s", params.Name)
	policyInput := policy.Input{
		Command:  command,
		ToolName: params.Name,
		CWD:      cwd,
	}
	llmEnabled := llm.Enabled(cwd)
	clf, clfErr := llm.ClassifierFor(cwd)
	if clfErr != nil {
		log.Printf("vibeguard llm: classifier init failed (fail-safe ask): %v", clfErr)
	}
	policyResult := policy.EvaluateWithLLM(ctx, cwd, policyInput, clf, llmEnabled)
	switch policyResult.Action {
	case policy.ActionAllow:
		return nil
	case policy.ActionDeny:
		msg := policyResult.Reason
		if msg == "" {
			msg = "tool call blocked by policy"
		}
		return fmt.Errorf("%s", msg)
	}

	created, err := client.RequestApproval(ctx, api.ApprovalRequest{
		Source:    "mcp",
		Client:    detectClient(),
		Command:   command,
		CWD:       cwd,
		ToolName:  params.Name,
		ToolInput: params.Arguments,
	})
	if err != nil {
		return fmt.Errorf("daemon unreachable: %w", err)
	}

	decision, err := client.WaitApproval(ctx, created.ID)
	if err != nil {
		return fmt.Errorf("approval wait failed: %w", err)
	}
	if decision.Permission != "allow" {
		msg := decision.UserMessage
		if msg == "" {
			msg = "tool call denied by user"
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}
