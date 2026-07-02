// EvaluateFull chains YAML policy, local detect, and optional LLM triage.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-3.0-integration.md).
package policy

import (
	"context"
	"strings"

	"github.com/alisaitteke/vibeguard/internal/approvalmode"
	"github.com/alisaitteke/vibeguard/internal/shell"
)

// EvaluateOpts configures the YAML → detect → optional LLM pipeline.
type EvaluateOpts struct {
	LLMEnabled bool
	Classifier Classifier
	Mode       approvalmode.Mode
}

// FullResult carries the final decision plus per-layer outcomes for audit.
type FullResult struct {
	Result
	YamlAction   Action
	DetectAction Action
	DetectRules  []string
	DetectScore  int
}

// EvaluateFull runs YAML Evaluate, then detect on shell IR, then optional LLM when
// detect returns ask and mode is Auto with LLM enabled. YAML and detect allow/deny
// short-circuit; AutoAllow/AutoDeny leave hook-level ask for daemon MaybeAutoDecide.
func EvaluateFull(ctx context.Context, cwd string, input Input, opts EvaluateOpts) FullResult {
	yaml := Evaluate(cwd, input)
	out := FullResult{
		Result:     yaml,
		YamlAction: yaml.Action,
	}

	if yaml.Action == ActionAllow || yaml.Action == ActionDeny {
		return out
	}

	ir := prepareIR(input)
	det, ok := runDetect(ir, input)
	if !ok {
		out.DetectAction = ActionAsk
		out.Result = Result{Action: ActionAsk, Reason: "detection engine unavailable"}
		return out
	}
	out.DetectAction = det.Action
	out.DetectRules = det.MatchedRules
	out.DetectScore = det.Score

	if det.Action == ActionAllow || det.Action == ActionDeny {
		out.Result = Result{Action: det.Action, Reason: det.Reason}
		return out
	}

	// Detect returned ask — mode gate and optional LLM.
	switch opts.Mode {
	case approvalmode.Auto:
		if opts.LLMEnabled && opts.Classifier != nil {
			out.Result = opts.Classifier.Classify(ctx, input, det.Reason)
			return out
		}
		out.Result = Result{Action: ActionAsk, Reason: det.Reason}
	case approvalmode.Ask, approvalmode.AutoAllow, approvalmode.AutoDeny:
		out.Result = Result{Action: ActionAsk, Reason: det.Reason}
	default:
		out.Result = Result{Action: ActionAsk, Reason: det.Reason}
	}
	return out
}

// prepareIR builds shell IR for detect. MCP-only inputs skip shell.Prepare so
// mcp_tool rules match on a minimal IR without parsing tool names as shell.
func prepareIR(input Input) shell.IR {
	cmd := strings.TrimSpace(input.Command)
	if cmd == "" {
		if input.ToolName != "" {
			return shell.IR{Raw: "mcp:" + input.ToolName}
		}
		return shell.IR{}
	}
	if strings.HasPrefix(cmd, "mcp:") && input.ToolName != "" {
		return shell.IR{Raw: cmd}
	}
	ir, _, _ := shell.Prepare(cmd)
	return ir
}
