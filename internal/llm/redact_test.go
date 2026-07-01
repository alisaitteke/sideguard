package llm

import (
	"strings"
	"testing"
)

func TestRedactCommand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		check func(t *testing.T, got string)
	}{
		{
			name:  "bearer token",
			input: `curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.xyz" api.example.com`,
			check: func(t *testing.T, got string) {
				if strings.Contains(got, "eyJhbGci") {
					t.Errorf("bearer token not redacted: %q", got)
				}
				if !strings.Contains(got, "Bearer [REDACTED]") {
					t.Errorf("want Bearer placeholder, got %q", got)
				}
			},
		},
		{
			name:  "sk token",
			input: "echo sk-abcdefghijklmnopqrstuvwxyz123456",
			check: func(t *testing.T, got string) {
				if strings.Contains(got, "abcdefghijklmnopqrstuvwxyz123456") {
					t.Errorf("sk token not redacted: %q", got)
				}
				if !strings.Contains(got, "sk-[REDACTED]") {
					t.Errorf("want sk placeholder, got %q", got)
				}
			},
		},
		{
			name:  "api_key assignment",
			input: "curl 'https://x.com?api_key=supersecretvalue'",
			check: func(t *testing.T, got string) {
				if strings.Contains(got, "supersecretvalue") {
					t.Errorf("api_key not redacted: %q", got)
				}
				if !strings.Contains(got, "[REDACTED]") {
					t.Errorf("want redaction placeholder, got %q", got)
				}
			},
		},
		{
			name:  "env path",
			input: "cat /home/user/project/.env.local",
			check: func(t *testing.T, got string) {
				if strings.Contains(got, ".env") {
					t.Errorf(".env path not redacted: %q", got)
				}
			},
		},
		{
			name:  "safe command unchanged",
			input: "git status",
			check: func(t *testing.T, got string) {
				if got != "git status" {
					t.Errorf("got %q, want unchanged git status", got)
				}
			},
		},
		{
			name:  "empty",
			input: "",
			check: func(t *testing.T, got string) {
				if got != "" {
					t.Errorf("got %q, want empty", got)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := RedactCommand(tc.input)
			tc.check(t, got)
		})
	}
}
