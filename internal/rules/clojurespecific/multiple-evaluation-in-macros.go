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
		return macroRuntimeSequence(node.Children[index+1:], parameter)
	}
	maximum := 0
	for _, arity := range node.Children[index:] {
		if arity == nil || arity.Type != reader.NodeList || len(arity.Children) < 2 ||
			arity.Children[0].Type != reader.NodeVector {
			continue
		}
		if count := macroRuntimeSequence(arity.Children[1:], parameter); count > maximum {
			maximum = count
		}
	}
	return maximum
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
		return macroRuntimeSequence(node.Children, parameter)
	}

	head := node.Children[0].Value
	// Many DSLs define declaration macros with names such as
	// `defprotocolpath` or `defrichnav`. Their arguments describe generated
	// vars and formal parameter vectors; without macro expansion their bodies
	// are not evidence that a caller expression executes twice.
	if strings.HasPrefix(head, "def") && head != "def" && head != "defonce" &&
		head != "defn" && head != "defn-" && head != "defmacro" && head != "defmethod" && head != "defmulti" {
		return 0
	}
	switch head {
	case "comment", "quote", "clojure.core/quote":
		return 0
	case "var":
		// var receives a symbol at macro expansion time; its argument is not
		// a caller expression evaluated by the generated code.
		return 0
	case "defn", "defn-", "defmacro":
		return macroGeneratedFunctionMax(node, parameter, true)
	case "fn", "fn*":
		return macroGeneratedFunctionMax(node, parameter, false)
	case "defprotocol", "extend-type", "extend-protocol", "defrecord", "deftype", "defmethod":
		// Generated methods/implementations are alternative dispatch paths;
		// do not add their bodies as if they executed sequentially.
		return macroRuntimeBranchMax(node.Children[1:], parameter)
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
		// always executed. Summing is a realizable worst-case path.
		return macroRuntimeSequence(node.Children[1:], parameter)
	default:
		return macroRuntimeSequence(node.Children, parameter)
	}
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
	index := 2
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

func macroQualifiedName(node *reader.RichNode, context map[string]interface{}) string {
	if node == nil || len(node.Children) < 2 || node.Children[1] == nil || node.Children[1].Type != reader.NodeSymbol {
		return ""
	}
	name := node.Children[1].Value
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
	parameters := multipleRuntimeParameters(node)
	if len(parameters) == 0 {
		return nil
	}
	if macroHasConfiguredEffectfulCall(context, macroQualifiedName(node, context), parameters) {
		return &rules.Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf(
				"The macro %s presents multiple calls to the input arguments %s without defining temporary local variables.",
				node.Children[1].Value, strings.Join(parameters, ", ")),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}
	return rules.SetContextualFindingWithEvidence(&rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"The macro %s presents multiple calls to the input arguments %s without defining temporary local variables.",
			node.Children[1].Value, strings.Join(parameters, ", ")),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}, "Repeating an expression in a macro may be part of the generated contract; the argument's purity and cost are not proven by the AST.", "generated-contract", "argument-purity", "argument-cost")
}

func init() {
	rules.RegisterRule(&MultipleEvaluationInMacrosRule{Rule: rules.Rule{
		ID:          "multiple-evaluation-in-macros",
		Name:        "Multiple Evaluation in Macros",
		Description: "Detects caller expressions inserted more than once on a realizable runtime path through generated macro code.",
		Severity:    rules.SeverityWarning,
	}})
}
