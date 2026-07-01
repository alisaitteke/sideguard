package llm

import (
	"errors"
	"testing"

	"github.com/alisaitteke/vibeguard/internal/policy"
)

func TestParseClassifyResponse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		action  policy.Action
		reason  string
		wantErr bool
	}{
		{
			name:   "plain json",
			input:  `{"action":"allow","reason":"safe"}`,
			action: policy.ActionAllow,
			reason: "safe",
		},
		{
			name:   "markdown fence",
			input:  "```json\n{\"action\":\"deny\",\"reason\":\"bad\"}\n```",
			action: policy.ActionDeny,
			reason: "bad",
		},
		{
			name:   "prose wrapped json",
			input:  `Here is the result: {"action":"ask","reason":"unclear"} thanks.`,
			action: policy.ActionAsk,
			reason: "unclear",
		},
		{
			name:   "uppercase action",
			input:  `{"action":"ALLOW","reason":"read-only"}`,
			action: policy.ActionAllow,
			reason: "read-only",
		},
		{
			name:   "empty reason allowed",
			input:  `{"action":"allow","reason":""}`,
			action: policy.ActionAllow,
			reason: "",
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid action",
			input:   `{"action":"maybe","reason":"x"}`,
			wantErr: true,
		},
		{
			name:    "garbage",
			input:   "not json at all",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseClassifyResponse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !isParseError(err) {
					t.Fatalf("expected parseError, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Action != tc.action {
				t.Errorf("action = %q, want %q", result.Action, tc.action)
			}
			if result.Reason != tc.reason {
				t.Errorf("reason = %q, want %q", result.Reason, tc.reason)
			}
		})
	}
}

func TestIsParseError(t *testing.T) {
	t.Parallel()

	if !isParseError(newParseError("test")) {
		t.Fatal("expected parseError to match isParseError")
	}
	if isParseError(errors.New("other")) {
		t.Fatal("unexpected match for generic error")
	}
}
