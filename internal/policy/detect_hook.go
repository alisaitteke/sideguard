// Detect registration breaks the policy↔detect import cycle: detect registers
// its evaluator at init time via SetDetectEvaluator.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-3.0-integration.md).
package policy

import "github.com/alisaitteke/sideguard/internal/shell"

// DetectOutcome is the detect layer result folded into EvaluateFull.
type DetectOutcome struct {
	Action       Action
	Reason       string
	MatchedRules []string
	Score        int
}

// DetectFunc evaluates shell IR through the detect engine.
type DetectFunc func(ir shell.IR, input Input) DetectOutcome

var detectEval DetectFunc

// SetDetectEvaluator registers the detect engine callback (called from detect.init).
func SetDetectEvaluator(fn DetectFunc) {
	detectEval = fn
}

func runDetect(ir shell.IR, input Input) (DetectOutcome, bool) {
	if detectEval == nil {
		return DetectOutcome{}, false
	}
	return detectEval(ir, input), true
}
