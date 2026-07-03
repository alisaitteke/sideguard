// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Package notify sends macOS alert-only notifications for pending approvals.
// Decision input stays in the terminal CLI (sideguard ui / approve / deny).
// See docs/plans/2026-07-01-0127-sideguard-foundation/ (vgf-phase-6.0-terminal-ui.md).
package notify

import (
	"fmt"
	"os"
	"strings"
)

const (
	maxBodyLen   = 120
	maxTitleLen  = 64
	notifyTitle  = "SideGuard"
	pendingHint  = " · Run: sideguard ui"
)

// envNotifications is the env var that enables desktop notifications when set to a truthy value.
// Default is disabled — set SIDEGUARD_NOTIFICATIONS=1 to enable.
const envNotifications = "SIDEGUARD_NOTIFICATIONS"

// NotificationsEnabled reports whether desktop notifications are enabled via SIDEGUARD_NOTIFICATIONS.
func NotificationsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envNotifications))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// PendingApproval notifies the user of a new pending approval (alert only).
// Long commands and secrets are truncated in the notification body.
// No-op when SIDEGUARD_NOTIFICATIONS is unset or not truthy (default: disabled).
func PendingApproval(id, client, command, toolName, source string) error {
	if !NotificationsEnabled() {
		return nil
	}
	body := formatBody(id, client, command, toolName, source)
	if err := sendMacOS(notifyTitle, body); err != nil {
		return fmt.Errorf("pending approval notification: %w", err)
	}
	return nil
}

// formatBody builds the notification line: "#shortId · Client · summary · Run: sideguard ui".
// The hint is always appended; summary is truncated so the full body fits maxBodyLen.
func formatBody(id, client, command, toolName, source string) string {
	shortID := shortApprovalID(id)
	clientLabel := formatClient(client)
	prefix := fmt.Sprintf("%s · %s · ", shortID, clientLabel)
	maxSummary := maxBodyLen - len(prefix) - len(pendingHint)
	if maxSummary < 8 {
		maxSummary = 8
	}
	summary := commandSummary(command, toolName, source)
	summary = truncate(summary, maxSummary)
	return prefix + summary + pendingHint
}

func shortApprovalID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "#?"
	}
	if len(id) > 8 {
		return "#" + id[:8]
	}
	return "#" + id
}

func formatClient(client string) string {
	client = strings.TrimSpace(strings.ToLower(client))
	switch client {
	case "cursor":
		return "Cursor"
	case "claude":
		return "Claude Code"
	case "":
		return "Agent"
	default:
		if len(client) == 0 {
			return "Agent"
		}
		return strings.ToUpper(client[:1]) + client[1:]
	}
}

func commandSummary(command, toolName, source string) string {
	if strings.TrimSpace(command) != "" {
		return truncate(command, maxBodyLen)
	}
	if strings.TrimSpace(toolName) != "" {
		if source == "mcp" {
			return "mcp:" + truncate(toolName, maxBodyLen-4)
		}
		return truncate(toolName, maxBodyLen)
	}
	return "approval required"
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
