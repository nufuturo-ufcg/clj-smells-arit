package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type NamespaceLoadSideEffectsRule struct {
	rules.Rule
}

func (r *NamespaceLoadSideEffectsRule) Meta() rules.Rule {
	return r.Rule
}

func (r *NamespaceLoadSideEffectsRule) checkRequire(symbol string) bool {
	
	if symbol == "require" || symbol == "requiring-resolve" {
		return true
	}

	return false
}

func (r *NamespaceLoadSideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	// Ignore build configuration files (project.clj, deps.edn) and standalone support scripts
	if strings.HasSuffix(filepath, "project.clj") || strings.HasSuffix(filepath, "deps.edn") || strings.Contains(filepath, "/support/") || strings.Contains(filepath, "/scripts/") {
		return nil
	}
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
				return &rules.Finding{
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
		Rule: rules.Rule{
			ID:          "namespace-load-side-effects",
			Name:        "Namespace Load Side Effercts",
			Description: "Using require operation outside a ns primary macro introduces hidden, dynamic dependencies that bypass the build tool's static dependency graph.",
			Severity:    rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}