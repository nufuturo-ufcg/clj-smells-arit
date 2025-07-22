package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

var mutatingSymbols = map[string]struct{}{
	"set!":           {},
	"alter-var-root": {},
	"agent-send":     {},
	"agent-send-off": {},
	"intern":         {},
	"aset":           {},
}

var contextualMutatingSymbols = map[string]struct{}{
	"ref-set": {},
	"reset!":  {},
}

var idiomaticMutatingSymbols = map[string]struct{}{
	"swap!": {},
}

var sideEffectSymbols = map[string]struct{}{
	"println": {},
	"print":   {},
	"prn":     {},
	"printf":  {},
	"spit":    {},
	"def":     {},
	"defonce": {},
	"intern":  {},
}

type ImmutabilityViolationRule struct {
	Rule
}

func (r *ImmutabilityViolationRule) Meta() Rule {
	return r.Rule
}

func (r *ImmutabilityViolationRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	isLocalScope := r.isInLocalScope(context)

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		symbol := node.Children[0]
		if symbol.Type == reader.NodeSymbol {

			if _, isMutating := mutatingSymbols[symbol.Value]; isMutating {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Found state mutation function call: `%s`. This can lead to side effects and violates immutability principles.", symbol.Value),
					Filepath: filepath,
					Location: symbol.Location,
					Severity: r.Severity,
				}
			}

			if _, isContextual := contextualMutatingSymbols[symbol.Value]; isContextual {
				finding := r.checkContextualMutation(symbol, context, filepath)
				if finding != nil {
					return finding
				}
			}

			if (symbol.Value == "def" || symbol.Value == "defonce") && isLocalScope {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Found `%s` inside a local scope. This mutates global state and should be avoided.", symbol.Value),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}

			if symbol.Value == "send" || symbol.Value == "send-off" {
				finding := r.checkAgentSideEffects(node, filepath)
				if finding != nil {
					return finding
				}
			}
		}
	}

	return nil
}

func (r *ImmutabilityViolationRule) isInLocalScope(context map[string]interface{}) bool {
	scopes := []string{"isInsideFunction", "isInsideLet", "isInsideLoop", "isInsideBinding"}

	for _, scope := range scopes {
		if val, ok := context[scope].(bool); ok && val {
			return true
		}
	}
	return false
}

func (r *ImmutabilityViolationRule) checkContextualMutation(symbol *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	switch symbol.Value {
	case "ref-set":

		if isInsideDosync, ok := context["isInsideDosync"].(bool); ok && isInsideDosync {
			return nil
		}
		return &Finding{
			RuleID:   r.ID,
			Message:  "Found `ref-set` outside of `dosync`. Use `dosync` to ensure transactional safety with refs.",
			Filepath: filepath,
			Location: symbol.Location,
			Severity: r.Severity,
		}
	case "reset!":

		return &Finding{
			RuleID:   r.ID,
			Message:  "Found `reset!`. Consider using `swap!` for atomic updates based on current value.",
			Filepath: filepath,
			Location: symbol.Location,
			Severity: SeverityInfo,
		}
	}
	return nil
}

func (r *ImmutabilityViolationRule) checkAgentSideEffects(node *reader.RichNode, filepath string) *Finding {

	if len(node.Children) >= 3 {
		fnArg := node.Children[2]
		if r.containsSideEffects(fnArg) {
			return &Finding{
				RuleID:   r.ID,
				Message:  "Found side effects in function passed to agent. Agent functions should be pure.",
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}
	return nil
}

func (r *ImmutabilityViolationRule) containsSideEffects(node *reader.RichNode) bool {
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		symbol := node.Children[0]
		if symbol.Type == reader.NodeSymbol {
			if _, hasSideEffect := sideEffectSymbols[symbol.Value]; hasSideEffect {
				return true
			}
		}
	}

	for _, child := range node.Children {
		if r.containsSideEffects(child) {
			return true
		}
	}

	return false
}

func init() {
	defaultRule := &ImmutabilityViolationRule{
		Rule: Rule{
			ID:          "immutability-violation",
			Name:        "Immutability Violation",
			Description: "Detects direct state mutation and violations of functional purity. Follows Clojure Style Guide recommendations for proper use of refs, atoms, agents, and avoiding global state mutation in local scopes.",
			Severity:    SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
}
