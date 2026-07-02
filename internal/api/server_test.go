package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if resp.Mode != "auto" {
		t.Fatalf("mode = %q, want auto", resp.Mode)
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

func TestIngestEventRoute(t *testing.T) {
	st := testStore(t)
	srv := NewServer("test", st)

	body := strings.NewReader(`{
		"source":"shell",
		"client":"cursor",
		"cwd":"/tmp",
		"command_redacted":"git status",
		"command_norm":"git status",
		"final_action":"allow",
		"decision_by":"detect"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/events", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	time.Sleep(100 * time.Millisecond)
	rows, err := st.QueryEvents(store.EventQuery{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].FinalAction != "allow" {
		t.Fatalf("final_action = %q, want allow", rows[0].FinalAction)
	}
}

func TestIngestEventRedactsSecret(t *testing.T) {
	st := testStore(t)
	srv := NewServer("test", st)

	body := strings.NewReader(`{
		"source":"shell",
		"client":"cursor",
		"cwd":"/tmp",
		"command_redacted":"export KEY=sk-abcdefghijklmnopqrstuvwxyz123456",
		"command_norm":"export KEY=sk-abcdefghijklmnopqrstuvwxyz123456",
		"final_action":"deny",
		"decision_by":"detect"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/events", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rec.Code)
	}

	time.Sleep(100 * time.Millisecond)
	rows, err := st.QueryEvents(store.EventQuery{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []struct {
		name, value string
	}{
		{"command_redacted", rows[0].CommandRedacted},
		{"command_norm", rows[0].CommandNorm},
	} {
		if strings.Contains(field.value, "sk-abcdefghijklmnopqrstuvwxyz123456") {
			t.Fatalf("secret leaked in %s: %q", field.name, field.value)
		}
	}
}

func TestQueryEventsRoute(t *testing.T) {
	st := testStore(t)
	srv := NewServer("test", st)

	for _, action := range []string{"allow", "deny"} {
		if err := st.IngestEvent(store.CommandEvent{
			Source:      "shell",
			Client:      "cursor",
			CWD:         "/tmp/work",
			FinalAction: action,
			DecisionBy:  "detect",
		}); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/events?denied=true&cwd=/tmp/work&limit=5", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var out []CommandEvent
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("events = %d, want 1", len(out))
	}
	if out[0].FinalAction != "deny" || out[0].CWD != "/tmp/work" {
		t.Fatalf("unexpected event: %+v", out[0])
	}
}

func TestQueryEventsBeforeRoute(t *testing.T) {
	st := testStore(t)
	srv := NewServer("test", st)

	base := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 4; i++ {
		if err := st.IngestEvent(store.CommandEvent{
			ID:          fmt.Sprintf("evt-%d", i),
			CreatedAt:   base.Add(time.Duration(i) * time.Second),
			Source:      "shell",
			Client:      "cursor",
			CWD:         "/tmp",
			FinalAction: "allow",
			DecisionBy:  "detect",
		}); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/events?limit=2", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("page1 status = %d body=%s", rec.Code, rec.Body.String())
	}
	var page1 []CommandEvent
	if err := json.NewDecoder(rec.Body).Decode(&page1); err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1 events = %d, want 2", len(page1))
	}

	before := page1[1].CreatedAt
	req2 := httptest.NewRequest(http.MethodGet, "/v1/events?limit=2&before="+before, nil)
	rec2 := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("page2 status = %d body=%s", rec2.Code, rec2.Body.String())
	}
	var page2 []CommandEvent
	if err := json.NewDecoder(rec2.Body).Decode(&page2); err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 {
		t.Fatalf("page2 events = %d, want 2", len(page2))
	}
	if page2[0].ID == page1[0].ID || page2[0].ID == page1[1].ID {
		t.Fatalf("page2 overlaps page1: page1=%v page2=%v", page1, page2)
	}

	reqBad := httptest.NewRequest(http.MethodGet, "/v1/events?before=not-a-date", nil)
	recBad := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("bad before status = %d, want 400", recBad.Code)
	}
}

func TestIngestEventMissingFields(t *testing.T) {
	srv := NewServer("test", testStore(t))
	body := strings.NewReader(`{"source":"shell"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/events", body)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
