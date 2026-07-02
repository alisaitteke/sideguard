// Package detect registers its evaluator with policy at init to avoid an import cycle.
package detect

import (
	"log"
	"sync"

	"github.com/alisaitteke/sideguard/internal/policy"
	"github.com/alisaitteke/sideguard/internal/shell"
)

var (
	regEngine     *Engine
	regEngineOnce sync.Once
	regEngineErr  error
)

func registeredEngine() (*Engine, error) {
	regEngineOnce.Do(func() {
		regEngine, regEngineErr = NewEngine()
	})
	return regEngine, regEngineErr
}

func init() {
	policy.SetDetectEvaluator(func(ir shell.IR, input policy.Input) policy.DetectOutcome {
		eng, err := registeredEngine()
		if err != nil {
			log.Printf("sideguard detect: engine init failed: %v", err)
			return policy.DetectOutcome{
				Action: policy.ActionAsk,
				Reason: "detection engine unavailable",
			}
		}
		r := eng.Evaluate(ir, input)
		return policy.DetectOutcome{
			Action:       r.Action,
			Reason:       r.Reason,
			MatchedRules: r.MatchedRules,
			Score:        r.Score,
		}
	})
}
