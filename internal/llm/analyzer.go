// Analyzer runs on-demand command analysis using the analysis signature.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-2.0-llm.md).
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alisaitteke/vibeguard/internal/config"
)

// AnalyzeResult is the structured output of on-demand command analysis.
type AnalyzeResult struct {
	Verdict     string // safe | caution | dangerous | unknown
	Summary     string
	Explanation string
	Provider    string // instance id used
}

// AnalyzeInput carries command context for analysis (command should be redacted by caller).
type AnalyzeInput struct {
	Command       string
	ToolName      string
	CWD           string
	ShellIR       string // optional enrichment summary
	DetectSummary string
}

// Analyzer performs user-initiated command safety analysis via LLM.
type Analyzer interface {
	Analyze(ctx context.Context, input AnalyzeInput) (AnalyzeResult, error)
}

type analyzer struct {
	driver    ChatDriver
	provider  string
	timeout   time.Duration
	signature string
}

// NewAnalyzer resolves the analysis provider and loads the analysis signature.
func NewAnalyzer(settings config.LLMSettings, creds map[string]config.ProviderCredential) (Analyzer, error) {
	providerID := settings.Analysis.Provider
	if providerID == "" {
		providerID = settings.DefaultProvider
	}

	instance, err := resolveProviderInstance(settings, providerID)
	if err != nil {
		return nil, err
	}

	cred, ok := creds[instance.ID]
	if !ok {
		cred = config.ProviderCredential{}
	}

	driver, err := NewChatDriver(instance, cred, settings.TimeoutMS)
	if err != nil {
		return nil, fmt.Errorf("analysis provider %q: %w", instance.ID, err)
	}

	sigName := settings.Analysis.Signature
	if sigName == "" {
		sigName = "analysis"
	}
	sig, err := LoadSignature(sigName)
	if err != nil {
		return nil, fmt.Errorf("load analysis signature %q: %w", sigName, err)
	}

	return &analyzer{
		driver:    driver,
		provider:  instance.ID,
		timeout:   time.Duration(settings.TimeoutMS) * time.Millisecond,
		signature: sig,
	}, nil
}

func (a *analyzer) Analyze(ctx context.Context, input AnalyzeInput) (AnalyzeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	redacted := RedactCommand(input.Command)
	shellIR := input.ShellIR
	if shellIR != "" {
		// Shell IR JSON embeds the raw command; redact before outbound LLM.
		shellIR = RedactCommand(shellIR)
	}
	userPrompt := buildAnalyzeUserMessage(AnalyzeInput{
		Command:       redacted,
		ToolName:      input.ToolName,
		CWD:           input.CWD,
		ShellIR:       shellIR,
		DetectSummary: input.DetectSummary,
	})

	content, err := a.driver.Chat(ctx, ChatRequest{
		SystemPrompt: a.signature,
		UserPrompt:   userPrompt,
	})
	if err != nil {
		return AnalyzeResult{}, err
	}

	parsed, parseErr := parseAnalyzeResponse(content)
	if parseErr != nil {
		return AnalyzeResult{
			Verdict:     "unknown",
			Summary:     "Could not parse LLM response",
			Explanation: parseErr.Error(),
			Provider:    a.provider,
		}, nil
	}

	return AnalyzeResult{
		Verdict:     parsed.Verdict,
		Summary:     parsed.Summary,
		Explanation: parsed.Explanation,
		Provider:    a.provider,
	}, nil
}

func buildAnalyzeUserMessage(input AnalyzeInput) string {
	payload := map[string]string{
		"command":   input.Command,
		"tool_name": input.ToolName,
		"cwd":       input.CWD,
	}
	if input.ShellIR != "" {
		payload["shell_ir"] = input.ShellIR
	}
	if input.DetectSummary != "" {
		payload["detect_summary"] = input.DetectSummary
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

type analyzeParsed struct {
	Verdict     string
	Summary     string
	Explanation string
}

func parseAnalyzeResponse(text string) (analyzeParsed, error) {
	text = stripMarkdownFence(text)
	text = extractFirstJSONObject(text)

	var parsed struct {
		Verdict     string `json:"verdict"`
		Summary     string `json:"summary"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return analyzeParsed{}, fmt.Errorf("parse analysis response: %w", err)
	}

	verdict := normalizeVerdict(parsed.Verdict)
	if verdict == "" {
		return analyzeParsed{}, fmt.Errorf("missing or invalid verdict in analysis response")
	}

	return analyzeParsed{
		Verdict:     verdict,
		Summary:     parsed.Summary,
		Explanation: parsed.Explanation,
	}, nil
}

func normalizeVerdict(v string) string {
	switch v {
	case "safe", "caution", "dangerous", "unknown":
		return v
	default:
		return ""
	}
}

// newAnalyzerWithDriver constructs an Analyzer for unit tests with an injected driver.
func newAnalyzerWithDriver(driver ChatDriver, providerID string, timeout time.Duration, signature string) Analyzer {
	return &analyzer{
		driver:    driver,
		provider:  providerID,
		timeout:   timeout,
		signature: signature,
	}
}
