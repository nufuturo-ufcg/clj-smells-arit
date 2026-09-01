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

func caseResolvesToLexicalBinding(n *reader.RichNode) bool {
	return n != nil && n.Resolution != nil &&
		n.Resolution.Kind == reader.ResolutionLocal &&
		n.Resolution.Lexical
}

// A top-level def is represented by the resolver as a non-lexical local
// resolution. It is distinct from a literal symbol because the reference is
// tied to a concrete def/defonce form in the current source file. That proves
// the case constant is not evaluated, but it does not prove that using the
// symbol was unintended; callers therefore receive a contextual finding.
func caseResolvesToLocalVarDefinition(n *reader.RichNode) bool {
	if n == nil || n.Resolution == nil || n.Resolution.Kind != reader.ResolutionLocal ||
		n.Resolution.Lexical || n.ResolvedDefinition == nil {
		return false
	}
	definition := n.ResolvedDefinition
	return definition.Type == reader.NodeList && len(definition.Children) > 1 &&
		definition.Children[0] != nil && definition.Children[0].Type == reader.NodeSymbol &&
		(definition.Children[0].Value == "def" || definition.Children[0].Value == "defonce") &&
		definition.Children[1] != nil && definition.Children[1].Type == reader.NodeSymbol &&
		definition.Children[1].Value == n.Value
}

func (r *CaseWithNonLiteralTestValuesRule) isNonLiteral(n *reader.RichNode) bool {
	if n == nil {
		return false
	}

	if n.Type == reader.NodeSymbol {
		// A symbol resolved to a local is the only source-level form this rule
		// can prove was intended as a runtime value.
		return caseResolvesToLexicalBinding(n)
	}

	switch n.Type {
	case reader.NodeKeyword, reader.NodeString, reader.NodeNumber,
		reader.NodeBool, reader.NodeNil, reader.NodeCharacter,
		reader.NodeList, reader.NodeVector, reader.NodeMap, reader.NodeSet,
		reader.NodeTag, reader.NodeSyntaxQuote, reader.NodeReaderCond,
		reader.NodeReaderCondSplice, reader.NodeReaderDiscard, reader.NodeQuote,
		reader.NodeVarQuote, reader.NodeUnquote, reader.NodeUnquoteSplice:
		// Lists group literal constants in `case`; collections, reader
		// conditionals, quoted forms, and discarded forms are compile-time data
		// or outside the source-level proof boundary.
		return false
	default:
		// Unknown reader or parser forms are not enough evidence for a finding.
		return false
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
	if firstElement.Type != reader.NodeSymbol || firstElement.Value != "case" ||
		caseResolvesToLexicalBinding(firstElement) {
		return nil
	}

	// Syntax: (case e clause1 clause2 ... default-clause?)
	// Clauses start at index 2. Each clause is a pair of (test-constant, result-expr).
	for i := 2; i < len(node.Children)-1; i += 2 {
		testNode := node.Children[i]
		if r.isNonLiteral(testNode) {
			// A resolved lexical local in a case constant position is not
			// evaluated. `case` compares the literal form, so this is a proven
			// semantic error rather than a contract-dependent warning.
			return &rules.Finding{
				RuleID:   r.Meta().ID,
				Message:  fmt.Sprintf("The `case` macro does not evaluate its test constants. Using the local symbol `%s` matches that literal symbol instead of its runtime value. Use `cond` or `condp` for dynamic evaluation.", testNode.Value),
				Filepath: filepath,
				Location: testNode.Location,
				Severity: r.Meta().Severity,
			}
		}
		if caseResolvesToLocalVarDefinition(testNode) {
			return rules.SetContextualFindingWithEvidence(&rules.Finding{
				RuleID:   r.Meta().ID,
				Message:  fmt.Sprintf("The `case` macro does not evaluate the local var `%s` used as a test constant. Review whether the literal symbol or its runtime value was intended; use a literal constant, `cond`, or `condp` as appropriate.", testNode.Value),
				Filepath: filepath,
				Location: testNode.Location,
				Severity: rules.ContextualSeverity(context, r.Meta().Severity),
				Tags:     rules.ContextualTags(context),
			}, "The var definition is resolved locally, but intent to match the symbol or its value is not provable from the AST.", "case-constant-intent", "var-value-contract")
		}
	}

	return nil
}

func init() {
	defaultRule := &CaseWithNonLiteralTestValuesRule{
		Rule: rules.Rule{
			ID:          "case-with-non-literal-test-values",
			Name:        "Case with Non-Literal Test Values",
			Description: "Flags resolved lexical symbols and source-local vars used as test values in `case` macros, which do not evaluate their test branches; ambiguous literals and dynamic expressions remain silent.",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}
