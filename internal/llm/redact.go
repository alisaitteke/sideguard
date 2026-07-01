// Redaction helpers strip sensitive substrings from commands before LLM prompts.
// See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-3.0-classifier.md).
package llm

import "regexp"

const redactedPlaceholder = "[REDACTED]"

var (
	bearerTokenRE   = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]+=*`)
	skTokenRE       = regexp.MustCompile(`sk-[A-Za-z0-9]{16,}`)
	apiKeyAssignRE  = regexp.MustCompile(`(?i)(api[_-]?key\s*=\s*)[^\s&'"]+`)
	envPathRE       = regexp.MustCompile(`\.env(?:\.[A-Za-z0-9_-]+)?`)
	highEntropyRE   = regexp.MustCompile(`\b[A-Za-z0-9+/]{40,}={0,2}\b`)
)

// RedactCommand replaces likely secrets in a shell command before it is sent to an LLM.
func RedactCommand(command string) string {
	if command == "" {
		return command
	}
	s := command
	s = bearerTokenRE.ReplaceAllString(s, "Bearer "+redactedPlaceholder)
	s = skTokenRE.ReplaceAllString(s, "sk-"+redactedPlaceholder)
	s = apiKeyAssignRE.ReplaceAllString(s, "${1}"+redactedPlaceholder)
	s = envPathRE.ReplaceAllString(s, redactedPlaceholder)
	s = highEntropyRE.ReplaceAllString(s, redactedPlaceholder)
	return s
}
