package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"
	"strings"

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
		mapIsInsideMacroDefinition(context) || r.IsInside(context, "__non-evaluated__", "comment") ||
		mapIsInCaseConstant(context) || mapIsTaggedLiteral(context) {
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
				return rules.SetContextualFindingWithEvidence(&rules.Finding{
					RuleID:   r.ID,
					Message:  "Explicit 'nil' value associated with a key in a map literal. Prefer omitting the key to represent absence of data.",
					Filepath: filepath,
					Location: valNode.Location,
					Severity: severity,
				}, "Explicit nil values in map literals may depend on external contracts, schemas, or serialization requirements.", "nil-value-contract", "fixed-shape-contract", "serialization-contract")
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
			var baseFields map[string]bool

			// Now arguments should be strictly in key-value pairs
			for i := 0; i < len(args)-1; i += 2 {
				valNode := args[i+1]
				if !r.isNilOrImplicitNil(valNode) {
					continue
				}
				if key := args[i]; key != nil && key.Type == reader.NodeKeyword {
					if baseFields == nil {
						baseFields = fileRecordFields(context)
					}
					// A record's base field cannot be omitted: dissoc would
					// turn the record into a plain map (e.g. a component's
					// stop). The target's type is rarely provable, so a base
					// field of any record of this file is excluded.
					if baseFields[strings.TrimPrefix(key.Value, ":")] {
						continue
					}
				}
				return rules.SetContextualFindingWithEvidence(&rules.Finding{
					RuleID:   r.ID,
					Message:  "Explicit 'nil' value associated with a key using 'assoc'. Prefer using 'cond->' or 'dissoc' instead of explicitly setting 'nil'.",
					Filepath: filepath,
					Location: valNode.Location,
					Severity: rules.SeverityHint,
				}, "Explicit nil values via assoc may depend on external contracts or map update protocols.", "nil-value-contract", "update-contract")
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

// fileRecordFields returns the base fields of every defrecord in the file,
// including records declared inside reader conditionals or other forms.
func fileRecordFields(context map[string]interface{}) map[string]bool {
	fields := map[string]bool{}
	var walk func(*reader.RichNode)
	walk = func(node *reader.RichNode) {
		if node == nil {
			return
		}
		if node.Type == reader.NodeList && len(node.Children) > 2 && node.Children[0] != nil &&
			strings.TrimPrefix(node.Children[0].Value, "clojure.core/") == "defrecord" {
			for _, child := range node.Children[2:] {
				if child != nil && child.Type == reader.NodeVector {
					for name := range symbolNames(child) {
						fields[name] = true
					}
					break
				}
			}
			return
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	roots, _ := context["file-roots"].([]*reader.RichNode)
	for _, root := range roots {
		walk(root)
	}
	return fields
}

func symbolNames(vector *reader.RichNode) map[string]bool {
	if vector == nil || vector.Type != reader.NodeVector {
		return nil
	}
	names := map[string]bool{}
	for _, child := range vector.Children {
		if child != nil && child.Type == reader.NodeSymbol {
			names[child.Value] = true
		}
	}
	return names
}

// mapIsTaggedLiteral reports a map read as the form of a tagged literal, such
// as a record literal #ns.Rec{:l nil} whose base fields cannot be omitted, or
// #js {...}; the reader function, not a map literal, defines its value.
func mapIsTaggedLiteral(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	return len(ancestors) > 0 && ancestors[len(ancestors)-1] != nil && ancestors[len(ancestors)-1].Type == reader.NodeTag
}

// mapIsInCaseConstant reports a case test constant: unevaluated literal data,
// like a quoted form. `(case x {:a nil} ...)` matches a map, it builds none.
func mapIsInCaseConstant(context map[string]interface{}) bool {
	value, _ := context["isInCaseConstantPosition"].(bool)
	return value
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
