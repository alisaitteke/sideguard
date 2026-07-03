// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package llm

import "encoding/json"

// buildUserMessage serializes classify input for the LLM user turn.
// Command redaction is applied by the classifier before the request reaches providers.
func buildUserMessage(req ClassifyRequest) string {
	payload := map[string]string{
		"command":     req.Input.Command,
		"tool_name":   req.Input.ToolName,
		"cwd":         req.Input.CWD,
		"yaml_reason": req.YAMLReason,
	}
	b, _ := json.Marshal(payload)
	return string(b)
}
