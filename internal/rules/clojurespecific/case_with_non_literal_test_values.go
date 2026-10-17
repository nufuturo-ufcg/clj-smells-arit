package clojurespecific

import (
	"fmt"
	"strings"

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
	// As a step of ->, some-> or cond->, the threaded value is the expression
	// and the clauses start at index 1.
	first := 2
	if caseIsThreadFirstStep(node, context) {
		first = 1
	}
	for i := first; i < len(node.Children)-1; i += 2 {
		testNode := node.Children[i]
		if r.isNonLiteral(testNode) {
			// A resolved lexical local in a case constant position is not
			// evaluated. `case` compares the literal form, so this is a proven
			// semantic error rather than a contract-dependent warning.
			return &rules.Finding{
				RuleID:     r.Meta().ID,
				Message:    fmt.Sprintf("The `case` macro does not evaluate its test constants. Using the local symbol `%s` matches that literal symbol instead of its runtime value. Use `cond` or `condp` for dynamic evaluation.", testNode.Value),
				Filepath:   filepath,
				Location:   testNode.Location,
				Severity:   r.Meta().Severity,
				Confidence: rules.ConfidenceProven,
			}
		}
		if finding := r.qualifiedTestFinding(testNode, filepath); finding != nil {
			return finding
		}
		// ClojureScript's case resolves symbols naming ^:const vars and
		// dispatches on their values (cljs.core/case, const?), so there the
		// symbol is a real constant. Only Clojure JVM matches the literal.
		if caseResolvesToLocalVarDefinition(testNode) && !(isClojureScriptSource(filepath) && caseVarIsConst(testNode)) {
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

// qualifiedTestFinding handles Java static members in a test position. On the
// JVM, `case` compares the literal symbol: Sequence/PPQ or DayOfWeek/SATURDAY
// never matches the field value. Qualified Clojure symbols (alias/name) are
// not reported: dispatching on symbol data such as 'my.ns/tag is common and
// intentional. ClojureScript is excluded because its case resolves ^:const vars.
func (r *CaseWithNonLiteralTestValuesRule) qualifiedTestFinding(testNode *reader.RichNode, filepath string) *rules.Finding {
	if testNode == nil || testNode.Type != reader.NodeSymbol || isClojureScriptSource(filepath) {
		return nil
	}
	slash := strings.Index(testNode.Value, "/")
	if slash <= 0 || slash == len(testNode.Value)-1 {
		return nil
	}
	prefix, member := testNode.Value[:slash], testNode.Value[slash+1:]
	kind := reader.ResolutionUnresolved
	if testNode.Resolution != nil {
		kind = testNode.Resolution.Kind
	}
	if kind == reader.ResolutionJavaStatic || (kind == reader.ResolutionUnresolved && looksLikeJavaConstant(prefix, member)) {
		return &rules.Finding{
			RuleID:     r.Meta().ID,
			Message:    fmt.Sprintf("The `case` macro does not evaluate its test constants: the Java static member `%s` is compared as a literal symbol and never matches the field value. Use `condp =` (or case on a literal such as the enum name).", testNode.Value),
			Filepath:   filepath,
			Location:   testNode.Location,
			Severity:   r.Meta().Severity,
			Confidence: rules.ConfidenceProven,
		}
	}
	return nil
}

func caseIsThreadFirstStep(node *reader.RichNode, context map[string]interface{}) bool {
	parent, _ := context["parent"].(*reader.RichNode)
	if parent == nil || parent.Type != reader.NodeList || len(parent.Children) < 3 ||
		parent.Children[0] == nil || parent.Children[0].Type != reader.NodeSymbol || parent.Children[1] == node {
		return false
	}
	switch strings.TrimPrefix(parent.Children[0].Value, "clojure.core/") {
	case "->", "some->":
		return true
	case "cond->":
		// (cond-> x test1 step1 test2 step2): steps sit at even indexes >= 3.
		for index := 3; index < len(parent.Children); index += 2 {
			if parent.Children[index] == node {
				return true
			}
		}
	}
	return false
}

// looksLikeJavaConstant matches the Java naming convention Class/FIELD or
// Class/EnumConstant when the class could not be resolved from imports.
func looksLikeJavaConstant(prefix, member string) bool {
	last := prefix
	if dot := strings.LastIndex(prefix, "."); dot >= 0 {
		last = prefix[dot+1:]
	}
	if last == "" || last[0] < 'A' || last[0] > 'Z' || member == "" || member[0] < 'A' || member[0] > 'Z' {
		return false
	}
	for _, ch := range member {
		if !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') && ch != '_' {
			return false
		}
	}
	return true
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

func isClojureScriptSource(filepath string) bool {
	return strings.HasSuffix(filepath, ".cljs") || strings.HasSuffix(filepath, ".cljc")
}

func caseVarIsConst(n *reader.RichNode) bool {
	if n == nil || n.ResolvedDefinition == nil || len(n.ResolvedDefinition.Children) < 2 {
		return false
	}
	name := n.ResolvedDefinition.Children[1]
	return name != nil && metadataContainsMarker(name.Metadata, ":const")
}
