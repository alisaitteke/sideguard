package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestHealthRoute(t *testing.T) {
	srv := NewServer("test", testStore(t))
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, want ok", resp.Status)
	}
}

func TestCreateApprovalRequest(t *testing.T) {
	srv := NewServer("test", testStore(t))
	body := strings.NewReader(`{"source":"shell","client":"cursor","command":"ls","cwd":"/tmp"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/approval/request", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}

	var resp ApprovalRequestResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "pending" || resp.ID == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestDecideAndWait(t *testing.T) {
	st := testStore(t)
	srv := NewServer("test", st)

	createBody := strings.NewReader(`{"source":"shell","client":"cursor","command":"echo hi","cwd":"/tmp"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/approval/request", createBody)
	createRec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d", createRec.Code)
	}
	var created ApprovalRequestResponse
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	done := make(chan struct{})
	var waitResp ApprovalDecisionResponse
	go func() {
		defer close(done)
		waitReq := httptest.NewRequest(http.MethodGet, "/v1/approval/"+created.ID+"/wait", nil)
		waitRec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(waitRec, waitReq)
		if waitRec.Code != http.StatusOK {
			t.Errorf("wait status = %d", waitRec.Code)
			return
		}
		_ = json.NewDecoder(waitRec.Body).Decode(&waitResp)
	}()

	decideBody := strings.NewReader(`{"decision":"allow"}`)
	decideReq := httptest.NewRequest(http.MethodPost, "/v1/approval/"+created.ID+"/decide", decideBody)
	decideRec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(decideRec, decideReq)
	if decideRec.Code != http.StatusOK {
		t.Fatalf("decide status = %d", decideRec.Code)
	}

	<-done
	if waitResp.Permission != "allow" {
		t.Fatalf("wait permission = %q, want allow", waitResp.Permission)
	}
}

func TestDecideUnknownID(t *testing.T) {
	srv := NewServer("test", testStore(t))
	body := strings.NewReader(`{"decision":"allow"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/approval/00000000-0000-0000-0000-000000000000/decide", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestGetApprovalModeDefault(t *testing.T) {
	srv := NewServer("test", testStore(t))
	req := httptest.NewRequest(http.MethodGet, "/v1/approval/mode", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp ApprovalModeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Mode != "ask" {
		t.Fatalf("mode = %q, want ask", resp.Mode)
	}
}

func TestSetApprovalModeInvalid(t *testing.T) {
	srv := NewServer("test", testStore(t))
	body := strings.NewReader(`{"mode":"bogus"}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/approval/mode", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreateApprovalAutoAllowMode(t *testing.T) {
	st := testStore(t)
	srv := NewServer("test", st)

	setBody := strings.NewReader(`{"mode":"auto_allow"}`)
	setReq := httptest.NewRequest(http.MethodPut, "/v1/approval/mode", setBody)
	setRec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(setRec, setReq)
	if setRec.Code != http.StatusOK {
		t.Fatalf("set mode status = %d body=%s", setRec.Code, setRec.Body.String())
	}

	createBody := strings.NewReader(`{"source":"shell","client":"cursor","command":"echo hi","cwd":"/tmp"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/approval/request", createBody)
	createRec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d", createRec.Code)
	}
	var created ApprovalRequestResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Status != "decided" {
		t.Fatalf("status = %q, want decided", created.Status)
	}

	waitReq := httptest.NewRequest(http.MethodGet, "/v1/approval/"+created.ID+"/wait", nil)
	waitRec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(waitRec, waitReq)
	if waitRec.Code != http.StatusOK {
		t.Fatalf("wait status = %d", waitRec.Code)
	}
	var waitResp ApprovalDecisionResponse
	if err := json.NewDecoder(waitRec.Body).Decode(&waitResp); err != nil {
		t.Fatal(err)
	}
	if waitResp.Permission != "allow" {
		t.Fatalf("permission = %q, want allow", waitResp.Permission)
	}
}

func TestServerBindsLoopback(t *testing.T) {
	srv := NewServer("test", testStore(t))
	if !IsLoopback(srv.Addr()) {
		t.Fatalf("expected loopback bind, got %q", srv.Addr())
	}
}
