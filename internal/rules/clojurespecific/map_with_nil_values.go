package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"

	"github.com/thlaurentino/arit/internal/reader"
)

type MapWithNilValuesRule struct {
	rules.Rule
}

func (r *MapWithNilValuesRule) Meta() rules.Rule {
	return r.Rule
}

func (r *MapWithNilValuesRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	if node == nil || rules.CurrentExecutionContext(context) == rules.ExecutionNonEvaluated ||
		rules.CurrentExecutionContext(context) == rules.ExecutionUnknown ||
		mapIsInsideMacroDefinition(context) || r.IsInside(context, "__non-evaluated__", "comment") {
		return nil
	}
	strict := rules.RuleSettingBool(context, r.Meta().ID, "strict", true)

	// 1. Check Map Literals: {:a nil}
	if node.Type == reader.NodeMap && !isDestructuringMapContext(node, context) {
		// children come in pairs: key, value, key, value
		for i := 1; i < len(node.Children); i += 2 {
			valNode := node.Children[i]
			if r.isNilOrImplicitNil(valNode) {
				severity := rules.SeverityHint
				if !strict {
					severity = rules.ContextualSeverity(context, r.Severity)
				}
				return &rules.Finding{
					RuleID:          r.ID,
					Message:         "Explicit 'nil' value associated with a key in a map literal. Prefer omitting the key to represent absence of data.",
					Filepath:        filepath,
					Location:        valNode.Location,
					Severity:        severity,
					MissingEvidence: []string{"nil-value-contract", "fixed-shape-contract", "serialization-contract"},
					Tags:            append(rules.ContextualTags(context), "contract-dependent"),
				}
			}
		}
	}

	// 2. Check assoc calls: (assoc m :a nil) or (assoc :a nil) inside ->
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		if isCoreAssocCall(node) && validAssocArity(node) {
			if isDestructuringMapContext(node, context) {
				return nil
			}
			args := node.Children[1:]
			args = args[1:]

			// Now arguments should be strictly in key-value pairs
			for i := 0; i < len(args)-1; i += 2 {
				valNode := args[i+1]
				if r.isNilOrImplicitNil(valNode) {
					return &rules.Finding{
						RuleID:          r.ID,
						Message:         "Explicit 'nil' value associated with a key using 'assoc'. Prefer using 'cond->' or 'dissoc' instead of explicitly setting 'nil'.",
						Filepath:        filepath,
						Location:        valNode.Location,
						Severity:        rules.SeverityHint,
						MissingEvidence: []string{"nil-value-contract", "update-contract"},
						Tags:            append(rules.ContextualTags(context), "low-confidence", "contract-dependent"),
					}
				}
			}
		}
	}

	return nil
}

func isDestructuringMapContext(node *reader.RichNode, context map[string]interface{}) bool {
	containsDestructuringMarker := func(candidate *reader.RichNode) bool {
		if candidate == nil || candidate.Type != reader.NodeMap {
			return false
		}
		for _, child := range candidate.Children {
			if child != nil && child.Type == reader.NodeKeyword {
				switch child.Value {
				case ":keys", ":strs", ":syms", ":or", ":as":
					return true
				}
			}
		}
		return false
	}
	if containsDestructuringMarker(node) {
		return true
	}
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if containsDestructuringMarker(ancestor) {
			return true
		}
	}
	parent, _ := context["parent"].(*reader.RichNode)
	if parent == nil || parent.Type != reader.NodeVector {
		return false
	}
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		switch ancestor.Children[0].Value {
		case "let", "loop", "binding", "if-let", "when-let", "for", "doseq", "defn", "defn-", "fn", "letfn":
			return true
		}
	}
	return false
}

func (r *MapWithNilValuesRule) isNilOrImplicitNil(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeNil {
		return true
	}
	return false
}

func isCoreAssocCall(node *reader.RichNode) bool {
	return rules.CallResolvesTo(node, "clojure.core/assoc") || rules.CallResolvesTo(node, "cljs.core/assoc")
}

func validAssocArity(node *reader.RichNode) bool {
	if node == nil || len(node.Children) < 4 {
		return false
	}
	return (len(node.Children)-2)%2 == 0
}

func mapIsInsideMacroDefinition(context map[string]interface{}) bool {
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
	defaultRule := &MapWithNilValuesRule{
		Rule: rules.Rule{
			ID:                    "map-with-nil-values",
			Name:                  "Map With Nil Values",
			Description:           "Detects the use of explicit nil values as values in map literals or assoc.",
			ContextualDescription: "It may be contextual in JSON responses, PATCH requests, schemas, ASTs, or fixed-shape maps. The finding remains visible with --include-contextual.",
			Severity:              rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}
