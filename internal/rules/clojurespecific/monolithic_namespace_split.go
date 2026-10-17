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
	if insideSyntaxQuote(context) {
		// Code template (e.g. an interpreter implementing ns): not executed here.
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
		if temporaryNamespaceSwitch(context) || insideFunctionBody(context) {
			// (try (in-ns 'x) ... (finally (in-ns saved))): evaluates in another
			// namespace and restores it; it does not continue a namespace.
			return nil
		}
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

func insideSyntaxQuote(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor != nil && ancestor.Type == reader.NodeSyntaxQuote {
			return true
		}
	}
	return false
}

// temporaryNamespaceSwitch reports whether the in-ns call belongs to a
// save/restore pattern: an enclosing try whose finally clause calls in-ns.
func temporaryNamespaceSwitch(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for i := len(ancestors) - 1; i >= 0; i-- {
		form := ancestors[i]
		if form == nil || form.Type != reader.NodeList || len(form.Children) == 0 ||
			form.Children[0] == nil || form.Children[0].Type != reader.NodeSymbol || form.Children[0].Value != "try" {
			continue
		}
		for _, clause := range form.Children[1:] {
			if clause != nil && clause.Type == reader.NodeList && len(clause.Children) > 1 &&
				clause.Children[0] != nil && clause.Children[0].Value == "finally" {
				for _, expr := range clause.Children[1:] {
					if rules.CallResolvesTo(expr, "clojure.core/in-ns") {
						return true
					}
				}
			}
		}
	}
	return false
}

func (r *MonolithicNamespaceSplitRule) finding(filepath string, node *reader.RichNode, message string, contextual bool) *rules.Finding {
	finding := &rules.Finding{
		RuleID:     r.ID,
		Message:    message,
		Filepath:   filepath,
		Location:   node.Location,
		Severity:   r.Severity,
		Confidence: rules.ConfidenceProven,
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

// insideFunctionBody reports an in-ns executed at call time rather than at
// load time: REPL tooling that evaluates code in another namespace and then
// switches back does not continue a namespace across files.
func insideFunctionBody(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, form := range ancestors {
		if form == nil {
			continue
		}
		if form.Type == reader.NodeFnLiteral {
			return true
		}
		if form.Type != reader.NodeList || len(form.Children) == 0 || form.Children[0] == nil ||
			form.Children[0].Type != reader.NodeSymbol {
			continue
		}
		switch strings.TrimPrefix(form.Children[0].Value, "clojure.core/") {
		case "defn", "defn-", "fn", "fn*", "defmethod", "letfn", "defmacro":
			return true
		}
	}
	return false
}
