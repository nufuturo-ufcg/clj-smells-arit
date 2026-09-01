package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type SingleSegmentNamespaceRule struct {
	rules.Rule
}

func (r *SingleSegmentNamespaceRule) Meta() rules.Rule {
	return r.Rule
}

func (r *SingleSegmentNamespaceRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if r.IsInside(context, "__non-evaluated__", "comment") ||
		rules.CurrentExecutionContext(context) == rules.ExecutionUnknown ||
		singleSegmentIsInsideMacroDefinition(context) {
		return nil
	}
	if node.Type == reader.NodeList && len(node.Children) >= 2 {
		if node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "ns" {
			if node.Children[1].Type == reader.NodeSymbol {
				nsName := node.Children[1].Value

				// 2. Extended safelist: allow single-segment namespaces that are standard in the ecosystem
				switch nsName {
				case "user", "dev", "test", "build", "repl", "script", "scratch":
					return nil
				}

				// 3. Main condition: missing a qualifying dot
				if !strings.Contains(nsName, ".") {
					return rules.SetContextualFindingWithEvidence(&rules.Finding{
						RuleID:   r.ID,
						Message:  fmt.Sprintf("Single-segment namespace '%s' detected. Prefer qualified namespaces (e.g. my-app.%s) to avoid collisions and tooling issues.", nsName, nsName),
						Filepath: filepath,
						Location: node.Location,
						Severity: rules.ContextualSeverity(context, r.Severity),
						Tags:     rules.ContextualTags(context),
					}, "Single-segment namespaces may be valid contracts in scripts, REPLs, examples, and small projects.", "namespace-scope-contract", "deployment-context")
				}
			}
		}
	}

	return nil
}

func singleSegmentIsInsideMacroDefinition(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		if ancestor.Children[0].Value == "defmacro" {
			return true
		}
	}
	return false
}

func init() {
	rules.RegisterRule(&SingleSegmentNamespaceRule{
		Rule: rules.Rule{
			ID:                    "single-segment-namespace",
			Name:                  "Single-segment namespace",
			Description:           "Detects namespaces declared with a single segment (ns foo) instead of qualified names (ns my-app.foo).",
			ContextualDescription: "It may be contextual in tests, fixtures, notebooks, benchmarks, scripts, and REPL namespaces. The finding remains visible with --include-contextual.",
			Severity:              rules.SeverityWarning,
		},
	})
}
