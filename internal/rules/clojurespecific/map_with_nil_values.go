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

	// 1. Check Map Literals: {:a nil}
	if node.Type == reader.NodeMap {
		// children come in pairs: key, value, key, value
		for i := 1; i < len(node.Children); i += 2 {
			valNode := node.Children[i]
			if r.isNilOrImplicitNil(valNode) {
				return &rules.Finding{
					RuleID:   r.ID,
					Message:  "Explicit 'nil' value associated with a key in a map literal. Prefer omitting the key to represent absence of data.",
					Filepath: filepath,
					Location: valNode.Location,
					Severity: r.Severity,
				}
			}
		}
	}

	// 2. Check assoc calls: (assoc m :a nil) or (assoc :a nil) inside ->
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		first := node.Children[0]
		if first != nil && first.Type == reader.NodeSymbol && (first.Value == "assoc" || first.Value == "clojure.core/assoc") {
			args := node.Children[1:]
			// If odd number of arguments, the first is the map. Skip it.
			if len(args)%2 != 0 {
				args = args[1:]
			}
			
			// Now arguments should be strictly in key-value pairs
			for i := 0; i < len(args)-1; i += 2 {
				valNode := args[i+1]
				if r.isNilOrImplicitNil(valNode) {
					return &rules.Finding{
						RuleID:   r.ID,
						Message:  "Explicit 'nil' value associated with a key using 'assoc'. Prefer using 'cond->' or 'dissoc' instead of explicitly setting 'nil'.",
						Filepath: filepath,
						Location: valNode.Location,
						Severity: r.Severity,
					}
				}
			}
		}
	}

	return nil
}

func (r *MapWithNilValuesRule) isNilOrImplicitNil(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeNil {
		return true
	}
	// Removed implicit nil checks for `when` and `if` because they generate excessive 
	// noise in real-world codebases where {:key (when cond val)} is widely accepted.
	return false
}

func init() {
	defaultRule := &MapWithNilValuesRule{
		Rule: rules.Rule{
			ID:          "map-with-nil-values",
			Name:        "Map With Nil Values",
			Description: "Detects the use of explicit nil values as values in map literals or assoc.",
			Severity:    rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}
