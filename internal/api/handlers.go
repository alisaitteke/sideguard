package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/alisaitteke/vibeguard/internal/notify"
	"github.com/alisaitteke/vibeguard/internal/store"
)

// Handler implements HTTP handlers for the daemon approval API.
// Long-poll wait mirrors git credential-cache blocking semantics.
// See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-2.0-daemon-core.md).
type Handler struct {
	Version string
	Store   *store.Store
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

	writeJSON(w, http.StatusAccepted, ApprovalRequestResponse{
		ID:     rec.ID,
		Status: "pending",
	})

	go notifyPendingApproval(rec)
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
