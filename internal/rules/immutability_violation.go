package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

var mutatingSymbols = map[string]struct{}{
	"reset!": {},

	"set!":           {},
	"alter-var-root": {},
	"agent-send":     {},
	"agent-send-off": {},
	"intern":         {},
	"ref-set":        {},

	"aset": {},
}

type ImmutabilityViolationRule struct {
	Rule
}

func (r *ImmutabilityViolationRule) Meta() Rule {
	return r.Rule
}

func (r *ImmutabilityViolationRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

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
		}
	}

	isInsideFunc, ok := context["isInsideFunction"].(bool)
	if !ok {
		isInsideFunc = false
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		symbol := node.Children[0]
		if symbol.Type == reader.NodeSymbol && (symbol.Value == "def" || symbol.Value == "defonce") && isInsideFunc {
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Found `%s` inside a function. This mutates global state and should be avoided.", symbol.Value),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

func init() {
	defaultRule := &ImmutabilityViolationRule{
		Rule: Rule{
			ID:          "immutability-violation",
			Name:        "Immutability Violation",
			Description: "Detects direct state mutation (e.g., reset!, swap!, set!, aset, etc.) or definition of global variables (def, defonce) inside functions. Both practices can lead to side effects and violate functional purity.",
			Severity:    SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
}
