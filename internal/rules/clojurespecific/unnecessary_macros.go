package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

type UnnecessaryMacrosRule struct{ rules.Rule }

type macroArityPart struct {
	params *reader.RichNode
	body   *reader.RichNode
}

func (r *UnnecessaryMacrosRule) Meta() rules.Rule { return r.Rule }

func macroContainsUnsafeFeature(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeUnquoteSplice {
		return true
	}
	if node.Type == reader.NodeNil {
		// A literal nil may be part of a macro's data contract; without expansion
		// and call-site evidence, keep this candidate silent.
		return true
	}
	if node.Type == reader.NodeSymbol {
		if strings.Contains(node.Value, "clojure.lang.RT") || strings.Contains(node.Value, "core.async") {
			return true
		}
		switch node.Value {
		case "&form", "&env", "gensym", "macroexpand", "macroexpand-1", "eval", "let", "let*", "loop", "recur", "fn", "fn*", "if", "if-not", "if-let", "if-some", "when", "when-not", "when-let", "when-some", "cond", "condp", "try", "catch", "finally", "binding", "set!", "case", "go", "go-loop", "thread", "slurp", "spit":
			return true
		}
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
		head := node.Children[0].Value
		// Primitive IFn implementations commonly use direct Java interop such
		// as `.invokePrim`; replacing the macro with a function can reintroduce
		// boxing or change the generated call shape.
		if strings.HasPrefix(head, ".") || strings.Contains(head, "invokePrim") || strings.Contains(head, "IFn") {
			return true
		}
	}
	for _, child := range node.Children {
		if macroContainsUnsafeFeature(child) {
			return true
		}
	}
	return false
}

func macroCountUnquotedParameters(node *reader.RichNode, parameters map[string]int, insideUnquote bool) {
	if node == nil {
		return
	}
	if node.Type == reader.NodeUnquote || node.Type == reader.NodeUnquoteSplice {
		for _, child := range node.Children {
			macroCountUnquotedParameters(child, parameters, true)
		}
		return
	}
	if insideUnquote && node.Type == reader.NodeSymbol {
		if _, exists := parameters[node.Value]; exists {
			parameters[node.Value]++
		}
	}
	for _, child := range node.Children {
		macroCountUnquotedParameters(child, parameters, insideUnquote)
	}
}

func macroDefinitionParts(node *reader.RichNode) (string, []macroArityPart, bool) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 4 {
		return "", nil, false
	}
	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol || head.Value != "defmacro" {
		return "", nil, false
	}
	if head.Resolution != nil {
		if head.Resolution.Kind == reader.ResolutionLocal {
			return "", nil, false
		}
		if head.Resolution.Kind != reader.ResolutionUnresolved &&
			(head.Resolution.Kind != reader.ResolutionClojureCore || head.Resolution.CanonicalName != "clojure.core/defmacro") {
			return "", nil, false
		}
	}
	name := node.Children[1]
	if name == nil || name.Type != reader.NodeSymbol {
		return "", nil, false
	}
	index := 2
	if index < len(node.Children) && node.Children[index].Type == reader.NodeString {
		index++
	}
	if index < len(node.Children) && node.Children[index].Type == reader.NodeMap {
		index++
	}
	if index >= len(node.Children) {
		return "", nil, false
	}

	if node.Children[index].Type == reader.NodeVector {
		if len(node.Children) != index+2 {
			return "", nil, false
		}
		return name.Value, []macroArityPart{{params: node.Children[index], body: node.Children[index+1]}}, true
	}

	// A multi-arity macro is safe for this rule only when every arity has one
	// body expression. Additional body forms may run at macro-expansion time
	// and are therefore outside the value-wrapper contract.
	parts := make([]macroArityPart, 0, len(node.Children)-index)
	for _, arity := range node.Children[index:] {
		if arity == nil || arity.Type != reader.NodeList || len(arity.Children) != 2 ||
			arity.Children[0] == nil || arity.Children[0].Type != reader.NodeVector || arity.Children[1] == nil {
			return "", nil, false
		}
		parts = append(parts, macroArityPart{params: arity.Children[0], body: arity.Children[1]})
	}
	if len(parts) == 0 {
		return "", nil, false
	}
	return name.Value, parts, true
}

func macroIsInsideLexicalBinding(context map[string]interface{}) bool {
	ancestors, ok := context["ancestorNodes"].([]*reader.RichNode)
	if !ok {
		return false
	}
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		switch ancestor.Children[0].Value {
		case "let", "let*", "loop", "loop*", "fn", "fn*", "letfn", "binding", "with-open":
			return true
		}
	}
	return false
}

func macroHasSyntaxQuote(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSyntaxQuote {
		return true
	}
	for _, child := range node.Children {
		if macroHasSyntaxQuote(child) {
			return true
		}
	}
	return false
}

// macroHasConfiguredValueSafeCallSites is deliberately opt-in. A local
// syntax-quote shape cannot prove that a macro is replaceable by a function:
// the macro receives forms, while a function receives evaluated values. The
// narrow promotion below requires an explicit macro allow-list and complete
// indexed call-site evidence for every observed invocation.
func macroHasConfiguredValueSafeCallSites(context map[string]interface{}, qualifiedName string) bool {
	if !rules.MatchesConfiguredName(qualifiedName, rules.RuleSettingStringSlice(context, "unnecessary-macros", "proven-macros")) {
		return false
	}
	index := rules.ProjectSemanticIndex(context)
	if index == nil || !index.MacroCallSitesComplete() {
		return false
	}
	definition, ok := index.MacroDefinitionOf(qualifiedName)
	if !ok {
		return false
	}
	calls := index.MacroCallSites(qualifiedName)
	if len(calls) == 0 {
		return false
	}
	coveredArities := make(map[int]bool, len(definition.Arities))
	for _, call := range calls {
		arityIndex, arity := macroArityIndexForCall(definition, len(call.Arguments))
		if arity == nil || len(call.ArgumentEvidence) != len(call.Arguments) {
			return false
		}
		coveredArities[arityIndex] = true
		for _, evidence := range call.ArgumentEvidence {
			if evidence.Evidence != semantics.EvidenceProven {
				return false
			}
		}
	}
	for arityIndex := range definition.Arities {
		if !coveredArities[arityIndex] {
			return false
		}
	}
	return len(definition.Arities) > 0
}

func (r *UnnecessaryMacrosRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	name, arities, ok := macroDefinitionParts(node)
	if !ok || macroIsInsideLexicalBinding(context) {
		return nil
	}
	for _, arity := range arities {
		if !macroHasSyntaxQuote(arity.body) || macroContainsUnsafeFeature(arity.body) {
			return nil
		}
		parameterUses := make(map[string]int)
		for _, param := range arity.params.Children {
			if param.Type == reader.NodeSymbol && param.Value != "&" {
				parameterUses[param.Value] = 0
			}
		}
		if len(parameterUses) == 0 {
			return nil
		}
		macroCountUnquotedParameters(arity.body, parameterUses, false)
		for _, uses := range parameterUses {
			if uses != 1 {
				return nil
			}
		}
	}
	finding := &rules.Finding{
		RuleID: r.ID, Filepath: filepath, Location: node.Location, Severity: r.Severity,
		Message: fmt.Sprintf("Macro %q only wraps function-like evaluation; prefer a normal function for composability.", name),
	}
	if macroHasConfiguredValueSafeCallSites(context, macroQualifiedName(node, context)) {
		return finding
	}
	return rules.SetContextualFindingWithEvidence(finding,
		"A simple macro may preserve syntax, compatibility, or a deliberate expansion API.",
		"explicit-macro-allow-list", "complete-indexed-call-sites", "proven-argument-evidence")
}

func init() {
	rules.RegisterRule(&UnnecessaryMacrosRule{Rule: rules.Rule{
		ID: "unnecessary-macros", Name: "Unnecessary Macros",
		Description: "Detects simple syntax-quoted macros with no compile-time or evaluation-control feature.",
		Severity:    rules.SeverityHint,
	}})
}
