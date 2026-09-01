package clojurespecific

import (
	"strings"

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
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
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
	switch rules.FileRole(context) {
	case "generated", "dev", "test", "build":
		return nil
	}
	namespace := strings.ToLower(rules.CurrentNamespace(context))
	if strings.HasPrefix(namespace, "clojure.") || strings.HasPrefix(namespace, "cljs.") ||
		strings.Contains(namespace, ".compiler") || strings.Contains(namespace, ".runtime") ||
		strings.HasSuffix(namespace, ".test") || strings.Contains(namespace, ".test-") {
		return nil
	}

	switch {
	case rules.CallResolvesTo(node, "clojure.core/load"):
		return r.finding(
			filepath,
			node,
			"Use of load stitches compilation from other files into this namespace and breaks static analysis and dependency tooling. Prefer separate namespaces and require.",
			true,
		)
	case rules.CallResolvesTo(node, "clojure.core/in-ns"):
		return r.finding(
			filepath,
			node,
			"Use of in-ns switches namespaces imperatively and is often used to continue a logical namespace across files. Prefer a proper ns form and require for each namespace.",
			true,
		)
	default:
		return nil
	}
}

func (r *MonolithicNamespaceSplitRule) finding(filepath string, node *reader.RichNode, message string, contextual bool) *rules.Finding {
	finding := &rules.Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
	if contextual {
		return rules.SetContextualFindingWithEvidence(finding, "An imperative namespace change may be deliberate in REPLs, plugins, and dynamic loading mechanisms.", "namespace-lifecycle", "dynamic-loading-contract")
	}
	return finding
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
