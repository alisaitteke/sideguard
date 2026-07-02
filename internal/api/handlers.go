package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alisaitteke/vibeguard/internal/approvalmode"
	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/llm"
	"github.com/alisaitteke/vibeguard/internal/notify"
	"github.com/alisaitteke/vibeguard/internal/store"
)

// Handler implements HTTP handlers for the daemon approval API.
// Long-poll wait mirrors git credential-cache blocking semantics.
// See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-2.0-daemon-core.md).
type Handler struct {
	Version string
	Store   *store.Store

	// NewAnalyzer overrides analyzer construction (tests only). Nil uses llm.NewAnalyzer.
	NewAnalyzer func(settings config.LLMSettings, creds map[string]config.ProviderCredential) (llm.Analyzer, error)
	// LoadLLMSettings overrides LLM config load (tests only). Nil uses config.LoadLLMSettings.
	LoadLLMSettings func(cwd string) (config.LLMSettings, error)
	// ResolveCredentials overrides credential load (tests only). Nil uses config.ResolveProviderCredentials.
	ResolveCredentials func() (map[string]config.ProviderCredential, error)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: h.Version,
	})
}

func (h *Handler) CreateApprovalRequest(w http.ResponseWriter, r *http.Request) {
	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Source == "" || req.Client == "" {
		writeError(w, http.StatusBadRequest, "source and client are required")
		return
	}

	rec, err := h.Store.CreateApproval(req.Source, req.Client, req.Command, req.CWD, req.ToolName, req.ToolInput)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create approval")
		return
	}

	status := "pending"
	if _, autoDecided, err := h.Store.MaybeAutoDecide(rec.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to auto-decide approval")
		return
	} else if autoDecided {
		status = "decided"
	}

	writeJSON(w, http.StatusAccepted, ApprovalRequestResponse{
		ID:     rec.ID,
		Status: status,
	})

	if status == "pending" {
		go notifyPendingApproval(rec)
	}
}

func (h *Handler) GetApprovalMode(w http.ResponseWriter, _ *http.Request) {
	mode, err := h.Store.GetApprovalMode()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get approval mode failed")
		return
	}
	writeJSON(w, http.StatusOK, ApprovalModeResponse{Mode: string(mode)})
}

func (h *Handler) SetApprovalMode(w http.ResponseWriter, r *http.Request) {
	var req SetApprovalModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	mode, err := approvalmode.Parse(req.Mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.SetApprovalMode(mode); err != nil {
		writeError(w, http.StatusInternalServerError, "set approval mode failed")
		return
	}
	writeJSON(w, http.StatusOK, ApprovalModeResponse{Mode: string(mode)})
}

func notifyPendingApproval(rec *store.ApprovalRecord) {
	if rec == nil {
		return
	}
	_ = notify.PendingApproval(rec.ID, rec.Client, rec.Command, rec.ToolName, rec.Source)
}

func (h *Handler) WaitApproval(w http.ResponseWriter, r *http.Request) {
	id := approvalIDFromPath(r.URL.Path, "/wait")
	if id == "" {
		writeError(w, http.StatusBadRequest, "approval id required")
		return
	}

	result, err := h.Store.WaitForDecision(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "wait failed")
		return
	}

	writeJSON(w, http.StatusOK, ApprovalDecisionResponse{
		Permission:   result.Permission,
		UserMessage:  result.UserMessage,
		AgentMessage: result.AgentMessage,
	})
}

func (h *Handler) DecideApproval(w http.ResponseWriter, r *http.Request) {
	id := approvalIDFromPath(r.URL.Path, "/decide")
	if id == "" {
		writeError(w, http.StatusBadRequest, "approval id required")
		return
	}

	var decision ApprovalDecision
	if err := json.NewDecoder(r.Body).Decode(&decision); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if decision.Decision != "allow" && decision.Decision != "deny" {
		writeError(w, http.StatusBadRequest, "decision must be allow or deny")
		return
	}

	result, changed, err := h.Store.DecideApproval(id, decision.Decision, decision.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already decided") {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "decide failed")
		return
	}

	status := http.StatusOK
	if !changed {
		status = http.StatusOK
	}

	writeJSON(w, status, ApprovalDecisionResponse{
		Permission:   result.Permission,
		UserMessage:  result.UserMessage,
		AgentMessage: result.AgentMessage,
	})
}

func (h *Handler) ListPending(w http.ResponseWriter, _ *http.Request) {
	records, err := h.Store.ListPending()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list pending failed")
		return
	}

	now := time.Now().UTC()
	out := make([]PendingApproval, 0, len(records))
	for _, rec := range records {
		out = append(out, PendingApproval{
			ID:         rec.ID,
			Source:     rec.Source,
			Client:     rec.Client,
			Command:    rec.Command,
			CWD:        rec.CWD,
			ToolName:   rec.ToolName,
			CreatedAt:  rec.CreatedAt.UTC().Format(time.RFC3339),
			AgeSeconds: int64(now.Sub(rec.CreatedAt).Seconds()),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) IngestEvent(w http.ResponseWriter, r *http.Request) {
	var req CommandEvent
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Source == "" || req.Client == "" || req.FinalAction == "" || req.DecisionBy == "" {
		writeError(w, http.StatusBadRequest, "source, client, final_action, and decision_by are required")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})

	go func() {
		if err := h.Store.IngestEvent(ToStoreEvent(req)); err != nil {
			log.Printf("vibeguard events: ingest failed: %v", err)
		}
	}()
}

func (h *Handler) QueryEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := EventQueryParams{
		Since:  q.Get("since"),
		Before: q.Get("before"),
		CWD:    q.Get("cwd"),
		Search: q.Get("search"),
	}
	if denied := q.Get("denied"); denied == "true" || denied == "1" {
		params.Denied = true
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			params.Limit = n
		}
	}

	storeQuery := store.EventQuery{
		Denied: params.Denied,
		CWD:    params.CWD,
		Limit:  params.Limit,
		Search: params.Search,
	}
	if params.Since != "" {
		t, err := time.Parse(time.RFC3339, params.Since)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid since timestamp")
			return
		}
		storeQuery.Since = t
	}
	if params.Before != "" {
		t, err := time.Parse(time.RFC3339, params.Before)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid before timestamp")
			return
		}
		storeQuery.Before = t
	}

	rows, err := h.Store.QueryEvents(storeQuery)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query events failed")
		return
	}

	out := make([]CommandEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, FromStoreEvent(row))
	}
	writeJSON(w, http.StatusOK, out)
}

// AnalyzeCommand runs on-demand LLM command analysis with shell/detect enrichment.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-3.0-api.md).
func (h *Handler) AnalyzeCommand(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	command := strings.TrimSpace(req.Command)
	toolName := strings.TrimSpace(req.ToolName)
	cwd := strings.TrimSpace(req.CWD)
	eventID := strings.TrimSpace(req.EventID)

	if eventID != "" {
		ev, err := h.Store.GetEventByID(eventID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "event lookup failed")
			return
		}
		if ev == nil {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		command = strings.TrimSpace(ev.CommandNorm)
		if command == "" {
			command = strings.TrimSpace(ev.CommandRedacted)
		}
		toolName = strings.TrimSpace(ev.ToolName)
		if cwd == "" {
			cwd = strings.TrimSpace(ev.CWD)
		}
	} else if command == "" {
		writeError(w, http.StatusBadRequest, "command or event_id is required")
		return
	}

	enrich := enrichForAnalyze(command, toolName, cwd)

	loadSettings := h.LoadLLMSettings
	if loadSettings == nil {
		loadSettings = config.LoadLLMSettings
	}
	settings, err := loadSettings(cwd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load llm settings failed")
		return
	}
	if !settings.Enabled {
		writeError(w, http.StatusServiceUnavailable, "llm analysis is disabled")
		return
	}
	if settings.DefaultProvider == "" && settings.Analysis.Provider == "" {
		writeError(w, http.StatusServiceUnavailable, "no llm provider configured")
		return
	}
	if len(settings.Providers) == 0 {
		writeError(w, http.StatusServiceUnavailable, "no llm provider configured")
		return
	}

	resolveCreds := h.ResolveCredentials
	if resolveCreds == nil {
		resolveCreds = config.ResolveProviderCredentials
	}
	creds, err := resolveCreds()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load provider credentials failed")
		return
	}

	newAnalyzer := h.NewAnalyzer
	if newAnalyzer == nil {
		newAnalyzer = llm.NewAnalyzer
	}
	analyzer, err := newAnalyzer(settings, creds)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "llm analyzer unavailable")
		return
	}

	redacted := llm.RedactCommand(command)
	if eventID != "" {
		log.Printf("vibeguard analyze: event_id=%s command=%q", eventID, redacted)
	} else {
		log.Printf("vibeguard analyze: command=%q", redacted)
	}

	result, err := analyzer.Analyze(r.Context(), llm.AnalyzeInput{
		Command:       command,
		ToolName:      toolName,
		CWD:           cwd,
		ShellIR:       enrich.ShellIR,
		DetectSummary: enrich.DetectSummary,
	})
	if err != nil {
		writeJSON(w, http.StatusOK, AnalyzeResponse{
			Verdict:      "unknown",
			Summary:      "Analysis unavailable",
			Explanation:  err.Error(),
			DetectAction: enrich.DetectAction,
			DetectRules:  enrich.DetectRules,
		})
		return
	}

	writeJSON(w, http.StatusOK, AnalyzeResponse{
		Verdict:      result.Verdict,
		Summary:      result.Summary,
		Explanation:  result.Explanation,
		Provider:     result.Provider,
		DetectAction: enrich.DetectAction,
		DetectRules:  enrich.DetectRules,
	})
}

func approvalIDFromPath(path, suffix string) string {
	path = strings.TrimPrefix(path, "/v1/approval/")
	return strings.TrimSuffix(path, suffix)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
