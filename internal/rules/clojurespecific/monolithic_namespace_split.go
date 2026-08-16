package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type MonolithicNamespaceSplitRule struct {
	rules.Rule
}

func (r *MonolithicNamespaceSplitRule) Meta() rules.Rule {
	return r.Rule
}

func (r *MonolithicNamespaceSplitRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node.Type != reader.NodeList || len(node.Children) < 1 {
		return nil
	}

	first := node.Children[0]
	if first.Type != reader.NodeSymbol {
		return nil
	}

	execution := rules.CurrentExecutionContext(context)
	if execution == rules.ExecutionNonEvaluated || execution == rules.ExecutionUnknown {
		return nil
	}

	switch {
	case rules.CallResolvesTo(node, "clojure.core/load"):
		return r.finding(
			filepath,
			node,
			"Use of load stitches compilation from other files into this namespace and breaks static analysis and dependency tooling. Prefer separate namespaces and require.",
		)
	case rules.CallResolvesTo(node, "clojure.core/in-ns"):
		return r.finding(
			filepath,
			node,
			"Use of in-ns switches namespaces imperatively and is often used to continue a logical namespace across files. Prefer a proper ns form and require for each namespace.",
		)
	default:
		return nil
	}
}

func (r *MonolithicNamespaceSplitRule) finding(filepath string, node *reader.RichNode, message string) *rules.Finding {
	return &rules.Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func init() {
	rules.RegisterRule(&MonolithicNamespaceSplitRule{
		Rule: rules.Rule{
			ID:   "monolithic-namespace-split",
			Name: "Monolithic Namespace Split",
			Description: "Detects imperative load and in-ns used to split a logical namespace across files. " +
				"These patterns break static analysis and explicit dependency resolution; prefer distinct namespaces with require.",
			Severity: rules.SeverityWarning,
		},
	})
}
