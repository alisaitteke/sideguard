// JSON response parsing for LLM classification output.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-3.0-classifier.md).
package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/policy"
)

// parseError marks failures to interpret model JSON; classifier maps these to ActionAsk.
type parseError struct {
	cause error
}

func (e *parseError) Error() string {
	return e.cause.Error()
}

func (e *parseError) Unwrap() error {
	return e.cause
}

func isParseError(err error) bool {
	var pe *parseError
	return errors.As(err, &pe)
}

func newParseError(format string, args ...any) error {
	return &parseError{cause: fmt.Errorf(format, args...)}
}

// parseClassifyResponse extracts {"action":"allow|deny|ask","reason":"..."} from model text.
func parseClassifyResponse(text string) (policy.Result, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return policy.Result{}, newParseError("empty LLM response")
	}

	text = stripMarkdownFence(text)
	text = extractFirstJSONObject(text)

	var parsed struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return policy.Result{}, newParseError("parse LLM response: %w", err)
	}

	action := policy.Action(strings.ToLower(strings.TrimSpace(parsed.Action)))
	switch action {
	case policy.ActionAllow, policy.ActionDeny, policy.ActionAsk:
		return policy.Result{Action: action, Reason: parsed.Reason}, nil
	default:
		return policy.Result{}, newParseError("invalid action %q", parsed.Action)
	}
}

func stripMarkdownFence(text string) string {
	if !strings.HasPrefix(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return text
	}
	start := 1
	end := len(lines)
	if strings.TrimSpace(lines[len(lines)-1]) == "```" {
		end = len(lines) - 1
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

// extractFirstJSONObject returns the first balanced {...} block in prose-wrapped model output.
func extractFirstJSONObject(text string) string {
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return text
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return text
}
