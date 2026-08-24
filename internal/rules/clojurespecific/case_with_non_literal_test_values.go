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
		// Symbols in a case constant position are normally literal symbols. A
		// symbol resolved to a local is the useful exception: it is almost
		// certainly being used as an evaluated value by mistake.
		return n.Resolution != nil && n.Resolution.Kind == reader.ResolutionLocal
	}

	if n.Type == reader.NodeList {
		// In `case`, a list is a grouping of constants, not an expression.
		for _, child := range n.Children {
			if r.isNonLiteral(child) {
				return true
			}
		}
		return false
	}

	switch n.Type {
	case reader.NodeKeyword, reader.NodeString, reader.NodeNumber,
		reader.NodeBool, reader.NodeNil, reader.NodeCharacter:
		return false
	case reader.NodeTag:
		// Tagged literals are read as one constant. The parser keeps the tag
		// and its value together under the tag node.
		if len(n.Children) == 0 {
			return false
		}
		for _, child := range n.Children {
			if r.isNonLiteral(child) {
				return true
			}
		}
		return false
	case reader.NodeVector, reader.NodeMap, reader.NodeSet:
		// Constant collections are valid case constants. Only collections
		// containing a runtime local or expression are non-literal.
		for _, child := range n.Children {
			if r.isNonLiteral(child) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func caseIsQuoted(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor != nil && ancestor.Type == reader.NodeSyntaxQuote {
			return true
		}
	}
	return false
}

func (r *CaseWithNonLiteralTestValuesRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}
	if caseIsQuoted(context) {
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
