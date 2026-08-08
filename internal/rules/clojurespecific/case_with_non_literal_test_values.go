package clojurespecific

import (
	"fmt"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type CaseWithNonLiteralTestValuesRule struct {
	rules.Rule
}

func (r *CaseWithNonLiteralTestValuesRule) Meta() rules.Rule {
	return r.Rule
}

func (r *CaseWithNonLiteralTestValuesRule) isNonLiteral(n *reader.RichNode) bool {
	if n == nil {
		return false
	}
	
	if n.Type == reader.NodeSymbol {
		if n.Value == "true" || n.Value == "false" || n.Value == "nil" {
			return false
		}
		// Any other symbol is treated as a literal symbol by `case`, but this usually indicates a bug
		// where the developer expected it to be evaluated as a variable.
		return true
	}

	if n.Type == reader.NodeList {
		// In `case`, a list is used to specify multiple test values.
		for _, child := range n.Children {
			if r.isNonLiteral(child) {
				return true
			}
		}
	}

	return false
}

func (r *CaseWithNonLiteralTestValuesRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol || firstElement.Value != "case" {
		return nil
	}

	// Syntax: (case e clause1 clause2 ... default-clause?)
	// Clauses start at index 2. Each clause is a pair of (test-constant, result-expr).
	for i := 2; i < len(node.Children)-1; i += 2 {
		testNode := node.Children[i]
		if r.isNonLiteral(testNode) {
			return &rules.Finding{
				RuleID:   r.Meta().ID,
				Message:  fmt.Sprintf("The `case` macro does not evaluate its test constants. Using a non-literal symbol or expression like `%s` will match the literal symbol itself, which is likely a bug. Use `cond` or `condp` for dynamic evaluation.", testNode.Value),
				Filepath: filepath,
				Location: testNode.Location,
				Severity: r.Meta().Severity,
			}
		}
	}

	return nil
}

func init() {
	defaultRule := &CaseWithNonLiteralTestValuesRule{
		Rule: rules.Rule{
			ID:          "case-with-non-literal-test-values",
			Name:        "Case with Non-Literal Test Values",
			Description: "Flags the usage of non-literal symbols or expressions as test values in `case` macros, which do not evaluate their test branches.",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}
