// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Event builders and conversions for command history ingest.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-4.0-history-store.md).
package api

import (
	"strings"
	"time"

	"github.com/alisaitteke/sideguard/internal/approvalmode"
	"github.com/alisaitteke/sideguard/internal/policy"
	"github.com/alisaitteke/sideguard/internal/shell"
	"github.com/alisaitteke/sideguard/internal/store"
)

// BuildCommandEvent constructs an API event from an intercept decision.
func BuildCommandEvent(req ApprovalRequest, fr policy.FullResult, finalAction, decisionBy, reason, approvalID string, latency time.Duration) CommandEvent {
	return CommandEvent{
		Source:          req.Source,
		Client:          req.Client,
		CWD:             req.CWD,
		CommandRedacted: req.Command,
		CommandNorm:     commandNorm(req),
		ToolName:        req.ToolName,
		YamlAction:      string(fr.YamlAction),
		DetectAction:    string(fr.DetectAction),
		DetectRules:     append([]string(nil), fr.DetectRules...),
		DetectScore:     fr.DetectScore,
		FinalAction:     finalAction,
		DecisionBy:      decisionBy,
		Reason:          reason,
		ApprovalID:      approvalID,
		LatencyMS:       latency.Milliseconds(),
	}
}

// InferDecisionBy maps policy layer outcomes to the persisted decision_by value.
func InferDecisionBy(fr policy.FullResult) string {
	switch {
	case fr.YamlAction == policy.ActionAllow || fr.YamlAction == policy.ActionDeny:
		return "yaml"
	case fr.DetectAction == policy.ActionAllow || fr.DetectAction == policy.ActionDeny:
		return "detect"
	case fr.DetectAction == policy.ActionAsk && (fr.Action == policy.ActionAllow || fr.Action == policy.ActionDeny):
		return "llm"
	default:
		return "detect"
	}
}

// QueueDecisionBy returns decision_by for daemon queue outcomes.
func QueueDecisionBy(mode approvalmode.Mode) string {
	if mode.Decision() != "" {
		return "mode"
	}
	return "user"
}

func commandNorm(req ApprovalRequest) string {
	cmd := strings.TrimSpace(req.Command)
	if cmd == "" {
		if req.ToolName != "" {
			return "mcp:" + req.ToolName
		}
		return ""
	}
	if strings.HasPrefix(cmd, "mcp:") && req.ToolName != "" {
		return cmd
	}
	ir, _, _ := shell.Prepare(cmd)
	if ir.Argv0 == "" {
		return shell.Normalize(cmd)
	}
	parts := make([]string, 0, 1+len(ir.Args))
	parts = append(parts, ir.Argv0)
	parts = append(parts, ir.Args...)
	return strings.Join(parts, " ")
}

// ToStoreEvent converts an API event to a store row.
func ToStoreEvent(e CommandEvent) store.CommandEvent {
	var createdAt time.Time
	if e.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, e.CreatedAt); err == nil {
			createdAt = t
		}
	}
	return store.CommandEvent{
		ID:              e.ID,
		CreatedAt:       createdAt,
		Source:          e.Source,
		Client:          e.Client,
		CWD:             e.CWD,
		CommandRedacted: e.CommandRedacted,
		CommandNorm:     e.CommandNorm,
		ToolName:        e.ToolName,
		YamlAction:      e.YamlAction,
		DetectAction:    e.DetectAction,
		DetectRules:     e.DetectRules,
		DetectScore:     e.DetectScore,
		FinalAction:     e.FinalAction,
		DecisionBy:      e.DecisionBy,
		Reason:          e.Reason,
		ApprovalID:      e.ApprovalID,
		LatencyMS:       e.LatencyMS,
	}
}

// FromStoreEvent converts a store row to the API JSON shape.
func FromStoreEvent(e store.CommandEvent) CommandEvent {
	return CommandEvent{
		ID:              e.ID,
		CreatedAt:       e.CreatedAt.UTC().Format(time.RFC3339),
		Source:          e.Source,
		Client:          e.Client,
		CWD:             e.CWD,
		CommandRedacted: e.CommandRedacted,
		CommandNorm:     e.CommandNorm,
		ToolName:        e.ToolName,
		YamlAction:      e.YamlAction,
		DetectAction:    e.DetectAction,
		DetectRules:     e.DetectRules,
		DetectScore:     e.DetectScore,
		FinalAction:     e.FinalAction,
		DecisionBy:      e.DecisionBy,
		Reason:          e.Reason,
		ApprovalID:      e.ApprovalID,
		LatencyMS:       e.LatencyMS,
	}
}
