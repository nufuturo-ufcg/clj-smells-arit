package clojurespecific

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

type MultipleEvaluationInMacrosRule struct {
	rules.Rule
}

func (r *MultipleEvaluationInMacrosRule) Meta() rules.Rule { return r.Rule }

func macroParameters(params *reader.RichNode) []string {
	if params == nil || params.Type != reader.NodeVector {
		return nil
	}
	result := make([]string, 0, len(params.Children))
	for _, param := range params.Children {
		if param == nil || param.Type != reader.NodeSymbol || param.Value == "&" || param.Value == "_" ||
			strings.HasPrefix(param.Value, ".") || strings.Contains(param.Value, "/") {
			continue
		}
		result = append(result, param.Value)
	}
	return result
}

func macroDirectUnquoteParameter(node *reader.RichNode, parameter string) bool {
	if node == nil || (node.Type != reader.NodeUnquote && node.Type != reader.NodeUnquoteSplice) || len(node.Children) != 1 {
		return false
	}
	child := node.Children[0]
	return child != nil && child.Type == reader.NodeSymbol && child.Value == parameter
}

func macroRuntimeSequence(nodes []*reader.RichNode, parameter string) int {
	total := 0
	for _, node := range nodes {
		total += macroRuntimeMax(node, parameter)
	}
	return total
}

func macroRuntimeBranchMax(nodes []*reader.RichNode, parameter string) int {
	maximum := 0
	for _, node := range nodes {
		if count := macroRuntimeMax(node, parameter); count > maximum {
			maximum = count
		}
	}
	return maximum
}

func macroBindingVectorMax(node *reader.RichNode, parameter string) int {
	if node == nil || node.Type != reader.NodeVector {
		return macroRuntimeMax(node, parameter)
	}
	for index := 0; index < len(node.Children); index += 2 {
		if macroDirectUnquoteParameter(node.Children[index], parameter) {
			return -1000
		}
	}
	total := 0
	for index := 1; index < len(node.Children); index += 2 {
		total += macroRuntimeMax(node.Children[index], parameter)
	}
	return total
}

func macroGeneratedFunctionMax(node *reader.RichNode, parameter string, named bool) int {
	index := 1
	if named && index < len(node.Children) {
		// defn/defmacro names are declarations, not evaluated expressions.
		index++
	} else if !named && index < len(node.Children) && node.Children[index].Type == reader.NodeSymbol {
		// Optional self-name in (fn name ...).
		index++
	}
	for index < len(node.Children) &&
		(node.Children[index].Type == reader.NodeString || node.Children[index].Type == reader.NodeMap) {
		index++
	}
	if index >= len(node.Children) {
		return 0
	}
	if node.Children[index].Type == reader.NodeVector {
		if macroParamsBindParameter(node.Children[index], parameter) {
			return -1000
		}
		return macroRuntimeSequence(node.Children[index+1:], parameter)
	}
	maximum := 0
	for _, arity := range node.Children[index:] {
		if arity == nil || arity.Type != reader.NodeList || len(arity.Children) < 2 ||
			arity.Children[0].Type != reader.NodeVector {
			continue
		}
		if macroParamsBindParameter(arity.Children[0], parameter) {
			return -1000
		}
		if count := macroRuntimeSequence(arity.Children[1:], parameter); count > maximum {
			maximum = count
		}
	}
	return maximum
}

// macroMethodImplementationsMax counts generated method implementations as
// alternative invocation paths. A method's parameter vector declares names and
// is never an evaluation of a caller expression; type names, field vectors, and
// protocol names are declarations as well.
func macroMethodImplementationsMax(nodes []*reader.RichNode, parameter string) int {
	maximum := 0
	for _, node := range nodes {
		if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 || node.Children[1] == nil {
			continue
		}
		if node.Children[1].Type == reader.NodeVector {
			if count := macroRuntimeSequence(node.Children[2:], parameter); count > maximum {
				maximum = count
			}
			continue
		}
		// Multi-arity method: (name ([params] body...) ([params] body...)).
		for _, arity := range node.Children[1:] {
			if arity == nil || arity.Type != reader.NodeList || len(arity.Children) < 1 ||
				arity.Children[0] == nil || arity.Children[0].Type != reader.NodeVector {
				continue
			}
			if count := macroRuntimeSequence(arity.Children[1:], parameter); count > maximum {
				maximum = count
			}
		}
	}
	return maximum
}

// macroParamsBindParameter reports whether a generated parameter vector uses
// the macro parameter as a binding name, e.g. (fn [~conn] ...): the caller
// supplies a symbol, so later ~conn occurrences are references, not repeated
// evaluations of a caller expression.
func macroParamsBindParameter(params *reader.RichNode, parameter string) bool {
	if params == nil || params.Type != reader.NodeVector {
		return false
	}
	for _, child := range params.Children {
		if macroDirectUnquoteParameter(child, parameter) {
			return true
		}
	}
	return false
}

// macroArityAlternativesMax handles generated forms whose head is not a
// literal symbol (e.g. (~defname ~name ([x] ...) ([x y] ...))): arity lists are
// alternative invocation paths, not sequential evaluations.
func macroArityAlternativesMax(node *reader.RichNode, parameter string) (int, bool) {
	arities := 0
	for _, child := range node.Children {
		if child != nil && child.Type == reader.NodeList && len(child.Children) >= 1 &&
			child.Children[0] != nil && child.Children[0].Type == reader.NodeVector {
			arities++
		}
	}
	if arities < 2 {
		return 0, false
	}
	prefix, maximum := 0, 0
	for _, child := range node.Children {
		if child != nil && child.Type == reader.NodeList && len(child.Children) >= 1 &&
			child.Children[0] != nil && child.Children[0].Type == reader.NodeVector {
			if macroParamsBindParameter(child.Children[0], parameter) {
				continue
			}
			if count := macroRuntimeSequence(child.Children[1:], parameter); count > maximum {
				maximum = count
			}
			continue
		}
		prefix += macroRuntimeMax(child, parameter)
	}
	return prefix + maximum, true
}

// macroRuntimeMax returns the maximum number of evaluations of one caller
// expression along a single runtime path through a generated template.
func macroRuntimeMax(node *reader.RichNode, parameter string) int {
	if node == nil {
		return 0
	}
	if macroDirectUnquoteParameter(node, parameter) {
		return 1
	}

	switch node.Type {
	case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeVarQuote, reader.NodeReaderDiscard:
		return 0
	case reader.NodeUnquote, reader.NodeUnquoteSplice:
		return 0
	case reader.NodeReaderCond:
		return macroRuntimeBranchMax(node.Children, parameter)
	}

	if node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0].Type != reader.NodeSymbol {
		if node.Type == reader.NodeList {
			if count, ok := macroArityAlternativesMax(node, parameter); ok {
				return count
			}
		}
		return macroRuntimeSequence(node.Children, parameter)
	}

	head := node.Children[0].Value
	// macrovich/case (net.cgrand.macrovich and copies such as
	// mount.tools.macrovich) selects one platform branch at expansion time:
	// (case :clj expr :cljs expr). Only one branch reaches runtime.
	if head == "macrovich/case" || strings.HasSuffix(head, ".macrovich/case") {
		maximum := 0
		for index := 2; index < len(node.Children); index += 2 {
			if count := macroRuntimeMax(node.Children[index], parameter); count > maximum {
				maximum = count
			}
		}
		return maximum
	}
	// Many DSLs define declaration macros with names such as
	// `defprotocolpath` or `defrichnav`. Their arguments describe generated
	// vars and formal parameter vectors; without macro expansion their bodies
	// are not evidence that a caller expression executes twice.
	if strings.HasPrefix(head, "def") && head != "def" && head != "defonce" &&
		head != "defn" && head != "defn-" && head != "defmacro" && head != "defmethod" && head != "defmulti" {
		return 0
	}
	// compile-if (and compile-when) choose a branch while the code is
	// compiled; the test runs at that time and only one branch reaches runtime.
	if head == "compile-if" || strings.HasSuffix(head, "/compile-if") {
		return macroRuntimeBranchMax(node.Children[2:], parameter)
	}
	switch head {
	case "comment", "quote", "clojure.core/quote":
		return 0
	case "declare", "clojure.core/declare":
		// declare receives var names, never caller expressions.
		return 0
	case "var":
		// var receives a symbol at macro expansion time; its argument is not
		// a caller expression evaluated by the generated code.
		return 0
	case "defn", "defn-", "defmacro":
		return macroGeneratedFunctionMax(node, parameter, true)
	case "fn", "fn*":
		return macroGeneratedFunctionMax(node, parameter, false)
	case "set!":
		// (set! place value): a symbol place names a field or var to assign;
		// it is not an evaluation of the caller's expression.
		if len(node.Children) != 3 {
			return macroRuntimeSequence(node.Children[1:], parameter)
		}
		place := 0
		if !macroDirectUnquoteParameter(node.Children[1], parameter) {
			place = macroRuntimeMax(node.Children[1], parameter)
		}
		return place + macroRuntimeMax(node.Children[2], parameter)
	case "defmethod":
		// (defmethod name dispatch params body...) or with arities: the
		// dispatch value runs at definition time and the body at call time;
		// the parameter vector declares locals and is never evaluated.
		if len(node.Children) < 4 {
			return 0
		}
		dispatch := macroRuntimeMax(node.Children[2], parameter)
		body := 0
		if node.Children[3] != nil && node.Children[3].Type == reader.NodeVector {
			body = macroRuntimeSequence(node.Children[4:], parameter)
		} else {
			for _, arity := range node.Children[3:] {
				if arity != nil && arity.Type == reader.NodeList && len(arity.Children) > 0 {
					if count := macroRuntimeSequence(arity.Children[1:], parameter); count > body {
						body = count
					}
				}
			}
		}
		if dispatch > body {
			return dispatch
		}
		return body
	case "defprotocol", "extend-type", "extend-protocol", "defrecord", "deftype":
		// Generated methods/implementations are alternative dispatch paths;
		// do not add their bodies as if they executed sequentially.
		return macroMethodImplementationsMax(node.Children[1:], parameter)
	case "reify":
		return macroMethodImplementationsMax(node.Children[1:], parameter)
	case "proxy":
		// (proxy [supers] [ctor-args] methods...): constructor arguments are
		// evaluated once; each method is an alternative invocation path.
		if len(node.Children) < 3 {
			return 0
		}
		return macroRuntimeMax(node.Children[2], parameter) +
			macroMethodImplementationsMax(node.Children[3:], parameter)
	case "def", "defonce":
		if len(node.Children) < 3 {
			return 0
		}
		return macroRuntimeMax(node.Children[len(node.Children)-1], parameter)
	case "let", "let*", "loop", "loop*", "binding", "with-open", "doseq", "for":
		if len(node.Children) < 2 {
			return 0
		}
		return macroBindingVectorMax(node.Children[1], parameter) +
			macroRuntimeSequence(node.Children[2:], parameter)
	case "catch":
		if len(node.Children) <= 3 {
			return 0
		}
		return macroRuntimeSequence(node.Children[3:], parameter)
	case "if", "if-not":
		if len(node.Children) < 3 {
			return macroRuntimeSequence(node.Children[1:], parameter)
		}
		return macroRuntimeMax(node.Children[1], parameter) +
			macroRuntimeBranchMax(node.Children[2:], parameter)
	case "if-cljs":
		// if-cljs selects one platform branch during macro expansion; its
		// branch forms are never evaluated sequentially at runtime.
		return macroRuntimeBranchMax(node.Children[1:], parameter)
	case "if-let", "if-some":
		if len(node.Children) < 3 {
			return macroRuntimeSequence(node.Children[1:], parameter)
		}
		return macroRuntimeMax(node.Children[1], parameter) +
			macroRuntimeBranchMax(node.Children[2:], parameter)
	case "when", "when-not", "when-let", "when-some":
		return macroRuntimeSequence(node.Children[1:], parameter)
	case "case":
		if len(node.Children) < 3 {
			return macroRuntimeSequence(node.Children[1:], parameter)
		}
		// Constants are not evaluated. Values occupy every second position;
		// an optional final default is also a possible branch.
		maximum := 0
		for index := 3; index < len(node.Children); index += 2 {
			if count := macroRuntimeMax(node.Children[index], parameter); count > maximum {
				maximum = count
			}
		}
		if len(node.Children)%2 == 1 {
			if count := macroRuntimeMax(node.Children[len(node.Children)-1], parameter); count > maximum {
				maximum = count
			}
		}
		return macroRuntimeMax(node.Children[1], parameter) + maximum
	case "cond":
		prefixTests := 0
		maximum := 0
		for index := 1; index+1 < len(node.Children); index += 2 {
			prefixTests += macroRuntimeMax(node.Children[index], parameter)
			path := prefixTests + macroRuntimeMax(node.Children[index+1], parameter)
			if path > maximum {
				maximum = path
			}
		}
		return maximum
	case "try":
		// A catch path can follow evaluation of the try body, and finally is
		// always executed, so the body, one catch, and finally form a
		// realizable worst-case path. At most one catch clause handles a given
		// exception: the clauses are alternatives, never a sequence.
		var body, catches []*reader.RichNode
		for _, child := range node.Children[1:] {
			if child != nil && child.Type == reader.NodeList && len(child.Children) > 0 &&
				child.Children[0].Type == reader.NodeSymbol && child.Children[0].Value == "catch" {
				catches = append(catches, child)
			} else {
				body = append(body, child)
			}
		}
		return macroRuntimeSequence(body, parameter) + macroRuntimeBranchMax(catches, parameter)
	default:
		// An unknown form that receives a binding-shaped vector with the
		// parameter in a name position may bind it, as let does; later
		// occurrences would then name the new local, not the caller's
		// expression. The expansion is ambiguous, so the count is suppressed.
		if !rules.CallResolvesTo(node, "clojure.core/"+strings.TrimPrefix(head, "clojure.core/")) {
			for _, child := range node.Children[1:] {
				if child != nil && child.Type == reader.NodeVector && (len(child.Children)%2 == 0 || macroContainsSplice(child)) {
					for index := 0; index < len(child.Children); index += 2 {
						if macroDirectUnquoteParameter(child.Children[index], parameter) {
							return -1000
						}
					}
				}
			}
		}
		// A variadic body spliced into a form of another namespace may feed a
		// DSL macro that compiles it to data, as (stevedore/script ~@body)
		// does; only the core and local forms are known to evaluate it.
		if strings.Contains(head, "/") && !strings.HasPrefix(head, "clojure.core/") && !macroJavaStaticHead(head) {
			total := 0
			for _, child := range node.Children {
				if child != nil && child.Type == reader.NodeUnquoteSplice && macroDirectUnquoteParameter(child, parameter) {
					continue
				}
				total += macroRuntimeMax(child, parameter)
			}
			return total
		}
		return macroRuntimeSequence(node.Children, parameter)
	}
}

// macroJavaStaticHead reports a Class/member head, whose arguments are always
// evaluated.
func macroJavaStaticHead(head string) bool {
	slash := strings.Index(head, "/")
	if slash <= 0 {
		return false
	}
	class := head[:slash]
	if dot := strings.LastIndex(class, "."); dot >= 0 {
		class = class[dot+1:]
	}
	return class != "" && class[0] >= 'A' && class[0] <= 'Z'
}

func macroContainsSplice(node *reader.RichNode) bool {
	for _, child := range node.Children {
		if child != nil && child.Type == reader.NodeUnquoteSplice {
			return true
		}
	}
	return false
}

func macroContainsSyntaxQuote(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSyntaxQuote {
		return true
	}
	for _, child := range node.Children {
		if macroContainsSyntaxQuote(child) {
			return true
		}
	}
	return false
}

// macroBindingFormBinds reports whether a binding vector introduces the
// symbol in a binding position, including inside destructuring forms.
func macroBindingFormBinds(bindings *reader.RichNode, symbol string) bool {
	if bindings == nil || bindings.Type != reader.NodeVector {
		return false
	}
	for index := 0; index < len(bindings.Children); index += 2 {
		if macroPatternBinds(bindings.Children[index], symbol) {
			return true
		}
	}
	return false
}

func macroPatternBinds(pattern *reader.RichNode, symbol string) bool {
	if pattern == nil {
		return false
	}
	if pattern.Type == reader.NodeSymbol {
		return pattern.Value == symbol
	}
	if pattern.Type == reader.NodeVector || pattern.Type == reader.NodeMap {
		for _, child := range pattern.Children {
			if macroPatternBinds(child, symbol) {
				return true
			}
		}
	}
	return false
}

// macroExpansionMax interprets only expansion-time control flow whose returned
// template is unambiguous. Unknown template-building calls cause suppression.
func macroExpansionMax(node *reader.RichNode, parameter string) (int, bool) {
	if node == nil {
		return 0, true
	}
	if node.Type == reader.NodeSyntaxQuote {
		if len(node.Children) != 1 {
			return 0, false
		}
		return macroRuntimeMax(node.Children[0], parameter), true
	}
	if node.Type == reader.NodeQuote || node.Type == reader.NodeVarQuote || node.Type == reader.NodeReaderDiscard {
		return 0, true
	}
	if node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0].Type != reader.NodeSymbol {
		if macroContainsSyntaxQuote(node) {
			return 0, false
		}
		return 0, true
	}

	head := node.Children[0].Value
	switch head {
	case "if", "if-not", "if-let", "if-some", "if-cljs":
		maximum := 0
		for _, branch := range node.Children[2:] {
			count, known := macroExpansionMax(branch, parameter)
			if !known {
				return 0, false
			}
			if count > maximum {
				maximum = count
			}
		}
		return maximum, true
	case "cond":
		maximum := 0
		for index := 2; index < len(node.Children); index += 2 {
			count, known := macroExpansionMax(node.Children[index], parameter)
			if !known {
				return 0, false
			}
			if count > maximum {
				maximum = count
			}
		}
		return maximum, true
	case "case":
		maximum := 0
		for index := 3; index < len(node.Children); index += 2 {
			count, known := macroExpansionMax(node.Children[index], parameter)
			if !known {
				return 0, false
			}
			if count > maximum {
				maximum = count
			}
		}
		return maximum, true
	case "let", "let*", "binding", "do":
		if len(node.Children) < 2 {
			return 0, true
		}
		if head != "do" && macroBindingFormBinds(node.Children[1], parameter) {
			// `(let [x (keyword x)] ...)` replaces the caller form during
			// expansion; later `~x` interpolate the computed value.
			return 0, false
		}
		return macroExpansionMax(node.Children[len(node.Children)-1], parameter)
	default:
		if macroContainsSyntaxQuote(node) {
			return 0, false
		}
		return 0, true
	}
}

func macroArities(node *reader.RichNode) []struct {
	params *reader.RichNode
	body   []*reader.RichNode
} {
	if node == nil || len(node.Children) < 3 {
		return nil
	}
	index := macroNameIndex(node) + 1
	for index < len(node.Children) &&
		(node.Children[index].Type == reader.NodeString || node.Children[index].Type == reader.NodeMap) {
		index++
	}
	if index >= len(node.Children) {
		return nil
	}
	if node.Children[index].Type == reader.NodeVector {
		return []struct {
			params *reader.RichNode
			body   []*reader.RichNode
		}{{node.Children[index], node.Children[index+1:]}}
	}

	result := make([]struct {
		params *reader.RichNode
		body   []*reader.RichNode
	}, 0)
	for _, arity := range node.Children[index:] {
		if arity != nil && arity.Type == reader.NodeList && len(arity.Children) >= 2 &&
			arity.Children[0].Type == reader.NodeVector {
			result = append(result, struct {
				params *reader.RichNode
				body   []*reader.RichNode
			}{arity.Children[0], arity.Children[1:]})
		}
	}
	return result
}

func multipleRuntimeParameters(node *reader.RichNode) []string {
	risky := make(map[string]struct{})
	for _, arity := range macroArities(node) {
		if len(arity.body) == 0 {
			continue
		}
		// A macro returns only its final body expression. Earlier expressions
		// run during expansion and cannot duplicate caller runtime evaluation.
		resultForm := arity.body[len(arity.body)-1]
		for _, parameter := range macroParameters(arity.params) {
			count, known := macroExpansionMax(resultForm, parameter)
			if known && count > 1 {
				risky[parameter] = struct{}{}
			}
		}
	}
	parameters := make([]string, 0, len(risky))
	for parameter := range risky {
		parameters = append(parameters, parameter)
	}
	sort.Strings(parameters)
	return parameters
}

// macroNameIndex returns the index of the declared name in a def-like form.
// The reader keeps `^:private` and `^{...}` metadata as keyword or map
// siblings before the name, so they are skipped.
func macroNameIndex(node *reader.RichNode) int {
	index := 1
	for index+1 < len(node.Children) && node.Children[index] != nil &&
		(node.Children[index].Type == reader.NodeKeyword || node.Children[index].Type == reader.NodeMap ||
			node.Children[index].Type == reader.NodeMetadata || node.Children[index].Type == reader.NodeTag) &&
		node.Children[index+1] != nil && node.Children[index+1].Type == reader.NodeSymbol {
		index++
	}
	return index
}

func macroDeclaredName(node *reader.RichNode) string {
	if node == nil {
		return ""
	}
	index := macroNameIndex(node)
	if index >= len(node.Children) || node.Children[index] == nil || node.Children[index].Type != reader.NodeSymbol {
		return ""
	}
	return node.Children[index].Value
}

func macroQualifiedName(node *reader.RichNode, context map[string]interface{}) string {
	name := macroDeclaredName(node)
	if name == "" {
		return ""
	}
	if namespace, ok := context["current-namespace"].(string); ok && namespace != "" {
		return namespace + "/" + name
	}
	return name
}

func macroArityForCall(definition semantics.MacroDefinition, argumentCount int) *semantics.MacroArity {
	_, arity := macroArityIndexForCall(definition, argumentCount)
	return arity
}

func macroArityIndexForCall(definition semantics.MacroDefinition, argumentCount int) (int, *semantics.MacroArity) {
	for index := range definition.Arities {
		arity := &definition.Arities[index]
		if arity.Variadic {
			minimum := len(arity.Parameters) - 1
			if argumentCount >= minimum {
				return index, arity
			}
			continue
		}
		if argumentCount == len(arity.Parameters) {
			return index, arity
		}
	}
	return -1, nil
}

func macroHasConfiguredEffectfulCall(context map[string]interface{}, qualifiedName string, parameters []string) bool {
	if !rules.MatchesConfiguredName(qualifiedName, rules.RuleSettingStringSlice(context, "multiple-evaluation-in-macros", "proven-macros")) {
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
	for _, call := range index.MacroCallSites(qualifiedName) {
		arity := macroArityForCall(definition, len(call.Arguments))
		if arity == nil {
			continue
		}
		for argumentIndex, evidence := range call.ArgumentEvidence {
			parameterIndex := argumentIndex
			if arity.Variadic && parameterIndex >= len(arity.Parameters)-1 {
				parameterIndex = len(arity.Parameters) - 1
			}
			if parameterIndex < 0 || parameterIndex >= len(arity.Parameters) {
				continue
			}
			parameter := arity.Parameters[parameterIndex]
			if !containsString(parameters, parameter) || evidence.Evidence != semantics.EvidenceProven || evidence.Effects == semantics.EffectNone {
				continue
			}
			return true
		}
	}
	return false
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func (r *MultipleEvaluationInMacrosRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if value, _ := context["isInCaseConstantPosition"].(bool); value {
		return nil
	}
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 ||
		node.Children[0].Type != reader.NodeSymbol || node.Children[0].Value != "defmacro" {
		return nil
	}
	parameters := withoutLocalMacroArguments(multipleRuntimeParameters(node), node, context)
	if len(parameters) == 0 {
		return nil
	}
	if macroHasConfiguredEffectfulCall(context, macroQualifiedName(node, context), parameters) {
		return &rules.Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf(
				"The macro %s presents multiple calls to the input arguments %s without defining temporary local variables.",
				macroDeclaredName(node), strings.Join(parameters, ", ")),
			Filepath:   filepath,
			Location:   node.Location,
			Severity:   r.Severity,
			Confidence: rules.ConfidenceProven,
		}
	}
	return rules.SetContextualFindingWithEvidence(&rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"The macro %s presents multiple calls to the input arguments %s without defining temporary local variables.",
			macroDeclaredName(node), strings.Join(parameters, ", ")),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}, "Repeating an expression in a macro may be part of the generated contract; the argument's purity and cost are not proven by the AST.", "generated-contract", "argument-purity", "argument-cost")
}

// fileMacros collects the macros defined anywhere in the file.
func fileMacros(context map[string]interface{}) map[string]*reader.RichNode {
	macros := map[string]*reader.RichNode{}
	roots, _ := context["file-roots"].([]*reader.RichNode)
	var walk func(node *reader.RichNode)
	walk = func(node *reader.RichNode) {
		if node == nil {
			return
		}
		if node.Type == reader.NodeList && len(node.Children) > 1 && node.Children[0] != nil &&
			node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "defmacro" {
			if name := macroDeclaredName(node); name != "" {
				macros[name] = node
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, root := range roots {
		walk(root)
	}
	return macros
}

// macroParameterIsExpansionOnly reports whether, in the arity of macro that
// accepts argCount arguments, the parameter at argIndex is used only while
// expanding (never interpolated as ~p into the generated code), e.g. an arity
// literal used to build argument vectors.
func macroParameterIsExpansionOnly(macro *reader.RichNode, argCount, argIndex int) bool {
	for _, arity := range macroArities(macro) {
		params := arity.params
		if params == nil || params.Type != reader.NodeVector || len(params.Children) != argCount {
			continue
		}
		for _, param := range params.Children {
			if param != nil && param.Type == reader.NodeSymbol && param.Value == "&" {
				return false
			}
		}
		param := params.Children[argIndex]
		if param == nil || param.Type != reader.NodeSymbol {
			return false
		}
		// Taint the parameter and expansion-time let bindings that carry the same
		// caller form ((let [sd (with-meta scan-d ...)] `(... ~sd ...))). Values
		// computed from it, such as (repeatedly n gensym), are generated code,
		// not the caller expression.
		tainted := map[string]bool{param.Value: true}
		containsTainted := func(n *reader.RichNode) bool {
			found := false
			var walk func(x *reader.RichNode)
			walk = func(x *reader.RichNode) {
				if x == nil || found {
					return
				}
				if x.Type == reader.NodeSymbol && tainted[x.Value] {
					found = true
					return
				}
				for _, c := range x.Children {
					walk(c)
				}
			}
			walk(n)
			return found
		}
		var collect func(n *reader.RichNode)
		collect = func(n *reader.RichNode) {
			if n == nil || n.Type == reader.NodeSyntaxQuote {
				return
			}
			if n.Type == reader.NodeList && len(n.Children) > 1 && n.Children[0] != nil &&
				n.Children[0].Type == reader.NodeSymbol && (n.Children[0].Value == "let" || n.Children[0].Value == "let*") &&
				n.Children[1] != nil && n.Children[1].Type == reader.NodeVector {
				bindings := n.Children[1].Children
				for k := 0; k+1 < len(bindings); k += 2 {
					if bindings[k] != nil && bindings[k].Type == reader.NodeSymbol && preservesTaint(bindings[k+1], tainted) {
						tainted[bindings[k].Value] = true
					}
				}
			}
			for _, c := range n.Children {
				collect(c)
			}
		}
		for _, form := range arity.body {
			collect(form)
		}
		interpolated := false
		var walk func(n *reader.RichNode)
		walk = func(n *reader.RichNode) {
			if n == nil || interpolated {
				return
			}
			if (n.Type == reader.NodeUnquote || n.Type == reader.NodeUnquoteSplice) && containsTainted(n) {
				interpolated = true
				return
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
		for _, form := range arity.body {
			walk(form)
		}
		last := arity.body[len(arity.body)-1]
		return !interpolated && !(last != nil && last.Type == reader.NodeSymbol && tainted[last.Value])
	}
	return false
}

// preservesTaint reports whether expr yields the same caller form as a tainted
// symbol: the symbol itself, (with-meta x ...), (vary-meta x ...), or an
// if/when whose result branches all preserve it.
func preservesTaint(expr *reader.RichNode, tainted map[string]bool) bool {
	if expr == nil {
		return false
	}
	if expr.Type == reader.NodeSymbol {
		return tainted[expr.Value]
	}
	if expr.Type != reader.NodeList || len(expr.Children) < 2 || expr.Children[0] == nil || expr.Children[0].Type != reader.NodeSymbol {
		return false
	}
	switch strings.TrimPrefix(expr.Children[0].Value, "clojure.core/") {
	case "with-meta", "vary-meta":
		return preservesTaint(expr.Children[1], tainted)
	case "if":
		if len(expr.Children) < 3 {
			return false
		}
		for _, branch := range expr.Children[2:] {
			if !preservesTaint(branch, tainted) {
				return false
			}
		}
		return true
	}
	return false
}

// withoutLocalMacroArguments drops parameters whose every occurrence in the
// generated template is an argument that another macro of the same file uses
// only at expansion time (e.g. the arity literal n in (curry ~n (lift-m ~n ~f))).
// Arguments the inner macro interpolates into its own code are still runtime
// evaluations and keep the parameter risky.
// macroParameterNeverEvaluated reports whether, in the fixed arity of macro
// that accepts argCount arguments, the parameter at argIndex is interpolated
// only where the generated code does not evaluate it, e.g. (quote ~p).
func macroParameterNeverEvaluated(macro *reader.RichNode, argCount, argIndex int) bool {
	for _, arity := range macroArities(macro) {
		params := arity.params
		if params == nil || params.Type != reader.NodeVector || len(params.Children) != argCount || len(arity.body) == 0 {
			continue
		}
		for _, param := range params.Children {
			if param != nil && param.Type == reader.NodeSymbol && param.Value == "&" {
				return false
			}
		}
		param := params.Children[argIndex]
		if param == nil || param.Type != reader.NodeSymbol {
			return false
		}
		count, known := macroExpansionMax(arity.body[len(arity.body)-1], param.Value)
		return known && count == 0
	}
	return false
}

func withoutLocalMacroArguments(parameters []string, node *reader.RichNode, context map[string]interface{}) []string {
	if len(parameters) == 0 {
		return parameters
	}
	macros := fileMacros(context)
	if len(macros) == 0 {
		return parameters
	}
	result := parameters[:0:0]
	for _, parameter := range parameters {
		onlyExpansionArgs, seen := true, false
		var walk func(n *reader.RichNode)
		walk = func(n *reader.RichNode) {
			if n == nil || !onlyExpansionArgs {
				return
			}
			// Quoted generated code is data: (quote ~p) evaluates nothing.
			if n.Type == reader.NodeQuote || (n.Type == reader.NodeList && len(n.Children) > 0 &&
				n.Children[0] != nil && n.Children[0].Type == reader.NodeSymbol &&
				(n.Children[0].Value == "quote" || n.Children[0].Value == "clojure.core/quote")) {
				return
			}
			if macroDirectUnquoteParameter(n, parameter) {
				// Reached outside a recognized macro-argument position.
				seen = true
				onlyExpansionArgs = false
				return
			}
			if n.Type == reader.NodeList && len(n.Children) > 1 && n.Children[0] != nil && n.Children[0].Type == reader.NodeSymbol {
				name := n.Children[0].Value
				if slash := strings.LastIndex(name, "/"); slash >= 0 {
					name = name[slash+1:]
				}
				if macro, ok := macros[name]; ok {
					for i, child := range n.Children[1:] {
						if macroDirectUnquoteParameter(child, parameter) {
							seen = true
							if !macroParameterIsExpansionOnly(macro, len(n.Children)-1, i) &&
								!macroParameterNeverEvaluated(macro, len(n.Children)-1, i) {
								onlyExpansionArgs = false
								return
							}
							continue
						}
						walk(child)
					}
					return
				}
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
		for _, arity := range macroArities(node) {
			if len(arity.body) > 0 {
				walk(arity.body[len(arity.body)-1])
			}
		}
		if !(seen && onlyExpansionArgs) {
			result = append(result, parameter)
		}
	}
	return result
}

func init() {
	rules.RegisterRule(&MultipleEvaluationInMacrosRule{Rule: rules.Rule{
		ID:          "multiple-evaluation-in-macros",
		Name:        "Multiple Evaluation in Macros",
		Description: "Detects caller expressions inserted more than once on a realizable runtime path through generated macro code.",
		Severity:    rules.SeverityWarning,
	}})
}
