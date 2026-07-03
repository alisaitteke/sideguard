// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// MCP JSON-RPC framing over STDIO (Content-Length headers).
// See docs/plans/2026-07-01-0127-sideguard-foundation/ (vgf-phase-4.0-mcp-proxy.md).
package proxy

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ReadFrame reads one MCP JSON-RPC message using Content-Length framing.
func ReadFrame(r *bufio.Reader) ([]byte, error) {
	var contentLength int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "content-length:") {
			value := strings.TrimSpace(line[len("content-length:"):])
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %q", line)
			}
			contentLength = n
		}
	}
	if contentLength <= 0 {
		return nil, fmt.Errorf("missing or zero Content-Length")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

// WriteFrame writes one MCP JSON-RPC message using Content-Length framing.
func WriteFrame(w io.Writer, payload []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
