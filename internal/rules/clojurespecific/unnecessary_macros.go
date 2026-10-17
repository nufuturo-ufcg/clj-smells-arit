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

// macroTemplateIsPlainCall reports whether a macro body is a syntax-quoted
// template that only builds ordinary runtime calls from its parameters. It is
// the local evidence required by the catalog contract: no expansion-time code,
// no evaluation control, no generated bindings, and no caller-specific syntax.
// Anything that cannot be verified keeps the rule silent.
func macroTemplateIsPlainCall(body *reader.RichNode, params map[string]int, context map[string]interface{}) bool {
	if body == nil || body.Type != reader.NodeSyntaxQuote || len(body.Children) != 1 {
		// Code outside the syntax quote runs during macro expansion.
		return false
	}
	return macroTemplateNodeIsPlain(body.Children[0], params, context)
}

func macroTemplateNodeIsPlain(node *reader.RichNode, params map[string]int, context map[string]interface{}) bool {
	if node == nil {
		return true
	}
	switch node.Type {
	case reader.NodeUnquote:
		// Only a bare parameter may be interpolated. `~(f ...)` computes at
		// expansion time and `~'sym` captures a caller-visible symbol.
		if len(node.Children) != 1 || node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol {
			return false
		}
		_, isParam := params[node.Children[0].Value]
		return isParam
	case reader.NodeQuote, reader.NodeVarQuote, reader.NodeFnLiteral, reader.NodeSyntaxQuote,
		reader.NodeReaderCond, reader.NodeReaderCondSplice, reader.NodeReaderEval, reader.NodeMetadata, reader.NodeTag:
		// Quoting a parameter, deferring evaluation, nested templates, and
		// reader transformations are macro-specific behavior.
		return !macroContainsUnquote(node)
	case reader.NodeList:
		if len(node.Children) == 0 {
			return true
		}
		if !macroTemplateHeadIsKnownFunction(node.Children[0], context) {
			return false
		}
		for _, child := range node.Children[1:] {
			if !macroTemplateNodeIsPlain(child, params, context) {
				return false
			}
		}
		return true
	}
	for _, child := range node.Children {
		if !macroTemplateNodeIsPlain(child, params, context) {
			return false
		}
	}
	return true
}

func macroContainsUnquote(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeUnquote || node.Type == reader.NodeUnquoteSplice {
		return true
	}
	for _, child := range node.Children {
		if macroContainsUnquote(child) {
			return true
		}
	}
	return false
}

// macroTemplateHeadIsKnownFunction accepts only call heads whose runtime
// meaning is a function call: clojure.core function vars, functions defined in
// the analyzed file, and Java constructors or static methods. Macros, special
// forms, unresolved symbols, and vars from other namespaces are unknown
// expansion semantics and are rejected.
func macroTemplateHeadIsKnownFunction(head *reader.RichNode, context map[string]interface{}) bool {
	if head != nil && head.Type == reader.NodeKeyword {
		// Keyword invocation is always a map lookup function call.
		return true
	}
	if head == nil || head.Type != reader.NodeSymbol {
		return false
	}
	if resolution := head.Resolution; resolution != nil {
		switch resolution.Kind {
		case reader.ResolutionClojureCore:
			_, ok := clojureCoreFunctions[strings.TrimPrefix(resolution.CanonicalName, "clojure.core/")]
			return ok
		case reader.ResolutionJavaConstructor:
			return strings.HasSuffix(head.Value, ".") && len(head.Value) > 1
		case reader.ResolutionJavaStatic:
			// The resolver also reports some referred vars as java-static, so
			// the symbol itself must have the `Class/member` shape.
			return macroSymbolIsJavaStatic(head.Value)
		case reader.ResolutionNamespaceVar:
			if _, ok := clojureLibraryFunctions[resolution.CanonicalName]; ok {
				return true
			}
			_, ok := rules.FunctionSummaries(context)[resolution.CanonicalName]
			return ok
		case reader.ResolutionUnresolved:
			// The analyzer's core table is partial. An unresolved core
			// function name is accepted only when the ns declaration proves
			// that clojure.core is referred normally and nothing else can
			// supply the name.
			name := strings.TrimPrefix(resolution.CanonicalName, "clojure.core/")
			if _, ok := clojureCoreFunctions[name]; !ok {
				return false
			}
			return nsReferCoreNameUnambiguously(context, name)
		case reader.ResolutionLocal:
			// File-level definitions resolve as non-lexical locals; only a
			// summarized defn is a function. Lexical locals are unknown.
			if resolution.Lexical {
				return false
			}
			_, ok := rules.FunctionSummaries(context)[resolution.CanonicalName]
			return ok
		default:
			return false
		}
	}
	name := head.Value
	if strings.HasSuffix(name, ".") && len(name) > 1 {
		return true
	}
	if _, local := rules.FunctionSummaries(context)[name]; local {
		return true
	}
	if _, core := clojureCoreFunctions[strings.TrimPrefix(name, "clojure.core/")]; core {
		// Without a resolution, a bare core name is trusted only when the
		// file does not define or exclude it; a local defmacro is not a
		// function summary and is rejected below by macroNameDefinedLocally.
		return !macroNameDefinedLocally(context, name)
	}
	return false
}

// nsReferCoreNameUnambiguously reports whether the file's ns form refers
// clojure.core/name without any mechanism that could replace it: no `:use`,
// no `:refer :all`, and no `:refer-clojure` clause (exclude, only, or rename).
func nsReferCoreNameUnambiguously(context map[string]interface{}, name string) bool {
	nsForm, _ := context["ns-form"].(*reader.RichNode)
	if nsForm == nil {
		return false
	}
	if _, local := rules.FunctionSummaries(context)[name]; local {
		return false
	}
	for _, clause := range nsForm.Children[1:] {
		if clause == nil || clause.Type != reader.NodeList || len(clause.Children) == 0 {
			continue
		}
		switch clause.Children[0].Value {
		case ":use", ":refer-clojure", ":load":
			return false
		case ":require":
			if nsClauseHasReferAll(clause) {
				return false
			}
		}
	}
	return true
}

func nsClauseHasReferAll(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	for index, child := range node.Children {
		if child != nil && child.Type == reader.NodeKeyword && child.Value == ":refer" &&
			index+1 < len(node.Children) && node.Children[index+1] != nil &&
			node.Children[index+1].Type == reader.NodeKeyword && node.Children[index+1].Value == ":all" {
			return true
		}
		if nsClauseHasReferAll(child) {
			return true
		}
	}
	return false
}

func macroSymbolIsJavaStatic(symbol string) bool {
	slash := strings.LastIndex(symbol, "/")
	if slash <= 0 || slash == len(symbol)-1 {
		return false
	}
	owner := symbol[:slash]
	simple := owner[strings.LastIndex(owner, ".")+1:]
	return simple != "" && simple[0] >= 'A' && simple[0] <= 'Z'
}

func macroNameDefinedLocally(context map[string]interface{}, name string) bool {
	index := rules.ProjectSemanticIndex(context)
	if index == nil {
		return false
	}
	for qualified := range index.Macros {
		if qualified == name || strings.HasSuffix(qualified, "/"+name) {
			return true
		}
	}
	return false
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
		if !macroTemplateIsPlainCall(arity.body, parameterUses, context) {
			return nil
		}
	}
	finding := &rules.Finding{
		RuleID: r.ID, Filepath: filepath, Location: node.Location, Severity: r.Severity,
		Confidence: rules.ConfidenceProven,
		Message:    fmt.Sprintf("Macro %q only wraps function-like evaluation; prefer a normal function for composability.", name),
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
