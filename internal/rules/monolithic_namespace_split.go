package rules

import (
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type MonolithicNamespaceSplitRule struct {
	Rule
}

func (r *MonolithicNamespaceSplitRule) Meta() Rule {
	return r.Rule
}

func (r *MonolithicNamespaceSplitRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) < 1 {
		return nil
	}

	first := node.Children[0]
	if first.Type != reader.NodeSymbol {
		return nil
	}

	base := symbolBaseName(first.Value)
	switch base {
	case "load":
		if isUnderCommentMacro(context) {
			return nil
		}
		return r.finding(
			filepath,
			node,
			"Use of load stitches compilation from other files into this namespace and breaks static analysis and dependency tooling. Prefer separate namespaces and require.",
		)
	case "in-ns":
		if isUnderCommentMacro(context) {
			return nil
		}
		return r.finding(
			filepath,
			node,
			"Use of in-ns switches namespaces imperatively and is often used to continue a logical namespace across files. Prefer a proper ns form and require for each namespace.",
		)
	default:
		return nil
	}
}

func (r *MonolithicNamespaceSplitRule) finding(filepath string, node *reader.RichNode, message string) *Finding {
	return &Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

// symbolBaseName returns the segment after the last slash (e.g. clojure.core/load -> load).
func symbolBaseName(sym string) string {
	if i := strings.LastIndex(sym, "/"); i >= 0 {
		return sym[i+1:]
	}
	return sym
}

func isUnderCommentMacro(context map[string]interface{}) bool {
	if context == nil {
		return false
	}
	raw, ok := context["parent"]
	if !ok {
		return false
	}
	parent, ok := raw.(*reader.RichNode)
	if !ok || parent == nil || parent.Type != reader.NodeList || len(parent.Children) < 1 {
		return false
	}
	head := parent.Children[0]
	if head.Type != reader.NodeSymbol {
		return false
	}
	return symbolBaseName(head.Value) == "comment"
}

func init() {
	RegisterRule(&MonolithicNamespaceSplitRule{
		Rule: Rule{
			ID:   "monolithic-namespace-split",
			Name: "Monolithic Namespace Split",
			Description: "Detects imperative load and in-ns used to split a logical namespace across files. " +
				"These patterns break static analysis and explicit dependency resolution; prefer distinct namespaces with require.",
			Severity: SeverityWarning,
		},
	})
}
