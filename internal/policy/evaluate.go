package policy

import (
	"log"
	"regexp"
)

// Evaluate loads policy for cwd and returns the decision. Precedence: deny > ask > allow.
// When no policy file exists, defaults to ActionAsk. Invalid YAML fails closed to deny.
func Evaluate(cwd string, input Input) Result {
	p, err := Load(cwd)
	if err != nil {
		log.Printf("vibeguard policy: load error (fail-closed deny): %v", err)
		return Result{
			Action:    ActionDeny,
			Reason:    "policy configuration error",
			LoadError: err,
		}
	}
	return p.Evaluate(input)
}

// Evaluate runs the match engine. When no rules match, defaults to ActionAsk.
func (p *Policy) Evaluate(input Input) Result {
	var (
		allowReason string
		askReason   string
		denyReason  string
		hasAllow    bool
		hasAsk      bool
		hasDeny     bool
	)

	for _, rule := range p.rules {
		if !rule.matches(input) {
			continue
		}
		switch rule.action {
		case ActionDeny:
			hasDeny = true
			if rule.reason != "" {
				denyReason = rule.reason
			}
		case ActionAsk:
			hasAsk = true
			if rule.reason != "" {
				askReason = rule.reason
			}
		case ActionAllow:
			hasAllow = true
			if rule.reason != "" {
				allowReason = rule.reason
			}
		}
	}

	if hasDeny {
		reason := denyReason
		if reason == "" {
			reason = "blocked by policy"
		}
		return Result{Action: ActionDeny, Reason: reason}
	}
	if hasAsk {
		reason := askReason
		return Result{Action: ActionAsk, Reason: reason}
	}
	if hasAllow {
		reason := allowReason
		return Result{Action: ActionAllow, Reason: reason}
	}

	return Result{Action: ActionAsk}
}

func (r compiledRule) matches(input Input) bool {
	if r.commandRe != nil && !r.commandRe.MatchString(input.Command) {
		return false
	}
	if r.mcpToolRe != nil && !r.mcpToolRe.MatchString(input.ToolName) {
		return false
	}
	if r.pathRe != nil && !pathMatches(r.pathRe, input) {
		return false
	}
	return true
}

func pathMatches(re *regexp.Regexp, input Input) bool {
	if input.CWD != "" && re.MatchString(input.CWD) {
		return true
	}
	if input.Command != "" && re.MatchString(input.Command) {
		return true
	}
	return false
}
