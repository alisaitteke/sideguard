//go:build darwin

// Analyse bridge helpers for the macOS tray detail view (no AppKit / CGO).
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-6.0-tray-analyse-darwin.md).
package darwin

import (
	"context"
	"strings"

	"github.com/alisaitteke/sideguard/internal/api"
)

// AnalyseResultJSON is the ObjC bridge payload pushed after POST /v1/analyze completes.
type AnalyseResultJSON struct {
	Verdict     string `json:"verdict"`
	Summary     string `json:"summary"`
	Explanation string `json:"explanation"`
	Provider    string `json:"provider"`
	Error       string `json:"error,omitempty"`
}

// BuildAnalyzeRequest maps tray detail context to the daemon analyze request.
// History rows pass useEventID=true so the daemon loads the stored event; pending rows
// send inline command text only.
func BuildAnalyzeRequest(rowID, command string, useEventID bool) api.AnalyzeRequest {
	if useEventID {
		if id := strings.TrimSpace(rowID); id != "" {
			return api.AnalyzeRequest{EventID: id}
		}
	}
	return api.AnalyzeRequest{Command: strings.TrimSpace(command)}
}

// RunAnalyze calls the daemon analyze endpoint and maps the response for the tray UI.
func RunAnalyze(ctx context.Context, client *api.Client, rowID, command string, useEventID bool) AnalyseResultJSON {
	req := BuildAnalyzeRequest(rowID, command, useEventID)
	if req.EventID == "" && req.Command == "" {
		return AnalyseResultJSON{Error: "No command to analyze"}
	}

	resp, err := client.Analyze(ctx, req)
	if err != nil {
		return AnalyseResultJSON{Error: SanitizeAnalyseError(err)}
	}

	return AnalyseResultJSON{
		Verdict:     resp.Verdict,
		Summary:     resp.Summary,
		Explanation: resp.Explanation,
		Provider:    resp.Provider,
	}
}

// SanitizeAnalyseError turns daemon/client failures into short user-facing tray messages.
func SanitizeAnalyseError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "daemon unreachable"):
		return "Daemon unreachable. Is SideGuard running?"
	case strings.Contains(lower, "llm analysis is disabled"):
		return "LLM analysis is disabled in config."
	case strings.Contains(lower, "no llm provider configured"):
		return "No LLM provider configured. Open Settings to add one."
	case strings.Contains(lower, "event not found"):
		return "Command event not found. Try again from history."
	default:
		return "Analysis failed. Try again or check daemon logs."
	}
}
