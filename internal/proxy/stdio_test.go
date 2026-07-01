package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
)

func TestStdioFrameRoundTrip(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	var buf bytes.Buffer
	if err := WriteFrame(&buf, payload); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}

	got, err := ReadFrame(bufio.NewReader(bytes.NewReader(buf.Bytes())))
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("round-trip mismatch:\nwant %s\ngot  %s", payload, got)
	}
}

func TestClassifyFrame(t *testing.T) {
	tests := []struct {
		method string
		want   Action
	}{
		{"initialize", ActionPassthrough},
		{"ping", ActionPassthrough},
		{"tools/list", ActionPassthrough},
		{"tools/call", ActionHoldApproval},
		{"notifications/initialized", ActionPassthrough},
	}

	for _, tc := range tests {
		frame, _ := json.Marshal(JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  tc.method,
		})
		action, _, err := ClassifyFrame(frame)
		if err != nil {
			t.Fatalf("%s: %v", tc.method, err)
		}
		if action != tc.want {
			t.Fatalf("%s: action = %v, want %v", tc.method, action, tc.want)
		}
	}
}

func TestBuildErrorResponse(t *testing.T) {
	payload, err := BuildErrorResponse(json.RawMessage(`42`), -32000, "denied")
	if err != nil {
		t.Fatalf("BuildErrorResponse: %v", err)
	}
	var msg JSONRPCMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(msg.ID) != "42" {
		t.Fatalf("id = %s, want 42", msg.ID)
	}
	if msg.Error == nil || msg.Error.Message != "denied" {
		t.Fatalf("unexpected error: %+v", msg.Error)
	}
}
