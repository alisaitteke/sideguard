package proxy

import (
	"encoding/json"
	"fmt"
	"log"
)

// Action describes how the proxy handles an inbound client frame.
type Action int

const (
	// ActionPassthrough forwards the frame to upstream unchanged.
	ActionPassthrough Action = iota
	// ActionHoldApproval blocks until the daemon approves or denies the call.
	ActionHoldApproval
)

// JSONRPCError is a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSONRPCMessage is a minimal JSON-RPC 2.0 envelope for MCP routing.
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// ToolsCallParams is the MCP tools/call params shape.
type ToolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ClassifyFrame inspects a client-originated JSON-RPC frame and returns the proxy action.
func ClassifyFrame(frame []byte) (Action, *JSONRPCMessage, error) {
	var msg JSONRPCMessage
	if err := json.Unmarshal(frame, &msg); err != nil {
		return ActionPassthrough, nil, nil
	}

	if msg.Method == "" {
		return ActionPassthrough, &msg, nil
	}

	switch msg.Method {
	case "tools/call":
		return ActionHoldApproval, &msg, nil
	case "tools/list":
		log.Printf("sideguard: tools/list pass-through")
		return ActionPassthrough, &msg, nil
	default:
		return ActionPassthrough, &msg, nil
	}
}

// ParseToolsCallParams extracts tool name and arguments from a tools/call message.
func ParseToolsCallParams(msg *JSONRPCMessage) (ToolsCallParams, error) {
	if msg == nil {
		return ToolsCallParams{}, fmt.Errorf("nil message")
	}
	var params ToolsCallParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return ToolsCallParams{}, fmt.Errorf("invalid tools/call params: %w", err)
	}
	if params.Name == "" {
		return ToolsCallParams{}, fmt.Errorf("tools/call missing tool name")
	}
	return params, nil
}

// BuildErrorResponse returns a JSON-RPC error response frame for the client.
func BuildErrorResponse(id json.RawMessage, code int, message string) ([]byte, error) {
	resp := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONRPCError{Code: code, Message: message},
	}
	return json.Marshal(resp)
}
