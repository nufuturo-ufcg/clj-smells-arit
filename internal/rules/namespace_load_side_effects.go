package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type NamespaceLoadSideEffectsRule struct {
	Rule
}

func (r *NamespaceLoadSideEffectsRule) Meta() Rule {
	return r.Rule
}

func (r *NamespaceLoadSideEffectsRule) checkRequire(symbol string) bool {
	
	if symbol == "require" || symbol == "requiring-resolve" {
		return true
	}

	return false
}

func (r *NamespaceLoadSideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		if r.checkRequire(node.Children[0].Value) {
			isInsideNs := false
			if enclosing, ok := context["enclosingForms"].([]string); ok {
				for _, f := range enclosing {
					if f == "ns" {
						isInsideNs = true
						break
					}
				}
			}

			if !isInsideNs {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Side effect: '%s' detected outside of ns macro.", node.Children[0].Value),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}
	return nil
}

func init() {
	defaultRule := &NamespaceLoadSideEffectsRule{
		Rule: Rule{
			ID:          "namespace-load-side-effects",
			Name:        "Namespace Load Side Effercts",
			Description: "Using require operation outside a ns primary macro introduces hidden, dynamic dependencies that bypass the build tool's static dependency graph.",
			Severity:    SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
}
