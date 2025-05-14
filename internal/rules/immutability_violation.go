package rules

import (
	"fmt"
	//"os"

	//"os"

	"github.com/thlaurentino/arit/internal/reader"
)

var mutatingSymbols = map[string]struct{}{
	"reset!":         {},
	"swap!":          {},
	"set!":           {},
	"alter-var-root": {},
	"agent-send":     {},
	"agent-send-off": {},
}

type ImmutabilityViolationRule struct {
	Rule
}

func (r *ImmutabilityViolationRule) Meta() Rule {
	return r.Rule
}

func (r *ImmutabilityViolationRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	nodeValForLog := node.Value
	if len(nodeValForLog) > 60 {
		nodeValForLog = nodeValForLog[:60] + "..."
	}
	//fmt.Fprintf(os.Stderr, "[DEBUG ImmutabilityRule INGRESS] Node Type: %s, Value: '%s', Children: %d, File: %s, Location: %s\n",
	//	node.Type, nodeValForLog, len(node.Children), filepath, node.Location)

	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		calledFunc := node.Children[0].Value
		if _, isMutating := mutatingSymbols[calledFunc]; isMutating {

			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Found direct state mutation function call: %s. This can lead to side effects and violates immutability principles.", calledFunc),
				Filepath: filepath,
				Location: node.Children[0].Location,
				Severity: r.Severity,
			}
		}
	}

	isInsideFunc, ok := context["isInsideFunction"].(bool)
	if !ok {
		isInsideFunc = false
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "def" {
		//fmt.Fprintf(os.Stderr, "[DEBUG ImmutabilityRule] Found (def ...) node at %s:%s. isInsideFunction context: %t\n", filepath, node.Location, isInsideFunc)
		if isInsideFunc {
			return &Finding{
				RuleID:   r.ID,
				Message:  "Found `def` usage inside a function. This mutates global state and should be avoided.",
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
			Description: "Detects direct state mutation (e.g., reset!, swap!, set!) or definition of global variables (`def`) inside functions. Both practices can lead to side effects and violate functional purity.",
			Severity:    SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
}
