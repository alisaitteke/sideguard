// Shell and detect enrichment helpers for POST /v1/analyze.
// See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-3.0-api.md).
package api

import (
	"encoding/json"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/detect"
	"github.com/alisaitteke/vibeguard/internal/policy"
	"github.com/alisaitteke/vibeguard/internal/shell"
)

type analyzeEnrichment struct {
	ShellIR       string
	DetectAction  string
	DetectRules   []string
	DetectSummary string
}

func enrichForAnalyze(command, toolName, cwd string) analyzeEnrichment {
	ir := prepareAnalyzeIR(command, toolName)
	shellIR := formatShellIR(ir)

	var out analyzeEnrichment
	out.ShellIR = shellIR

	engine, err := detect.NewEngine()
	if err != nil {
		return out
	}

	det := engine.Evaluate(ir, policy.Input{
		Command:  command,
		ToolName: toolName,
		CWD:      cwd,
	})
	out.DetectAction = string(det.Action)
	out.DetectRules = append([]string(nil), det.MatchedRules...)
	out.DetectSummary = formatDetectSummary(det)
	return out
}

func prepareAnalyzeIR(command, toolName string) shell.IR {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		if toolName != "" {
			return shell.IR{Raw: "mcp:" + toolName}
		}
		return shell.IR{}
	}
	if strings.HasPrefix(cmd, "mcp:") && toolName != "" {
		return shell.IR{Raw: cmd}
	}
	ir, _, _ := shell.Prepare(cmd)
	return ir
}

func formatShellIR(ir shell.IR) string {
	if ir.Raw == "" && ir.Argv0 == "" {
		return ""
	}
	payload := map[string]any{
		"raw":   ir.Raw,
		"argv0": ir.Argv0,
	}
	if len(ir.Args) > 0 {
		payload["args"] = ir.Args
	}
	if len(ir.Stages) > 1 {
		stages := make([]map[string]any, 0, len(ir.Stages))
		for _, st := range ir.Stages {
			stages = append(stages, map[string]any{
				"argv0": st.Argv0,
				"args":  st.Args,
			})
		}
		payload["stages"] = stages
	}
	if len(ir.Substitutions) > 0 {
		payload["substitutions"] = ir.Substitutions
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

func formatDetectSummary(r detect.Result) string {
	payload := map[string]any{
		"action": string(r.Action),
		"reason": r.Reason,
		"score":  r.Score,
	}
	if len(r.MatchedRules) > 0 {
		payload["rules"] = r.MatchedRules
	}
	if len(r.Categories) > 0 {
		cats := make([]string, 0, len(r.Categories))
		for _, c := range r.Categories {
			cats = append(cats, string(c))
		}
		payload["categories"] = cats
	}
	b, _ := json.Marshal(payload)
	return string(b)
}
