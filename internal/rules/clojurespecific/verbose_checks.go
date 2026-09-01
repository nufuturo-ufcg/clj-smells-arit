package clojurespecific

import (
	"fmt"
	"strings"
	"sync"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

func isCoreVerboseCall(node *reader.RichNode, name string) bool {
	return rules.CallResolvesTo(node, "clojure.core/"+name)
}

func hasProvenCallShape(node *reader.RichNode, context map[string]interface{}) bool {
	_, proven := rules.ProvenCallFacts(node, context, "")
	return proven
}

func isCoreReplacementAvailable(context map[string]interface{}, name string) bool {
	shadowed, _ := context["shadowed-core"].(map[string]bool)
	if shadowed[name] {
		return false
	}
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestorBindsName(ancestor, name) {
			return false
		}
	}
	return true
}

func suggestedCoreName(suggestion string) string {
	suggestion = strings.TrimPrefix(suggestion, "(")
	if index := strings.IndexByte(suggestion, ' '); index >= 0 {
		return suggestion[:index]
	}
	return strings.TrimSuffix(suggestion, ")")
}

func ancestorBindsName(node *reader.RichNode, name string) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 ||
		node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol {
		return false
	}
	head := node.Children[0].Value
	switch head {
	case "defn", "defn-":
		index := 2
		if index < len(node.Children) && node.Children[index] != nil && node.Children[index].Type == reader.NodeString {
			index++
		}
		if index < len(node.Children) && node.Children[index] != nil && node.Children[index].Type == reader.NodeMap {
			index++
		}
		return functionParamsBindName(node, index, name)
	case "fn", "fn*":
		return functionParamsBindName(node, 1, name)
	case "let", "let*", "loop", "loop*", "binding", "with-open", "with-local-vars", "doseq", "for":
		if len(node.Children) > 1 {
			return bindingVectorBindsName(node.Children[1], name)
		}
	case "letfn":
		if len(node.Children) > 1 && node.Children[1] != nil && node.Children[1].Type == reader.NodeVector {
			for _, binding := range node.Children[1].Children {
				if binding != nil && binding.Type == reader.NodeList && len(binding.Children) > 0 &&
					binding.Children[0] != nil && binding.Children[0].Type == reader.NodeSymbol && binding.Children[0].Value == name {
					return true
				}
			}
		}
	case "catch":
		return len(node.Children) > 2 && node.Children[2] != nil && node.Children[2].Type == reader.NodeSymbol && node.Children[2].Value == name
	}
	return false
}

func functionParamsBindName(node *reader.RichNode, index int, name string) bool {
	if index >= len(node.Children) || node.Children[index] == nil {
		return false
	}
	params := node.Children[index]
	if params.Type == reader.NodeVector {
		return bindingSymbolsContainName(params, name)
	}
	if params.Type == reader.NodeList {
		for _, arity := range params.Children {
			if arity != nil && arity.Type == reader.NodeList && len(arity.Children) > 0 &&
				arity.Children[0] != nil && arity.Children[0].Type == reader.NodeVector &&
				bindingSymbolsContainName(arity.Children[0], name) {
				return true
			}
		}
	}
	return false
}

func bindingVectorBindsName(node *reader.RichNode, name string) bool {
	if node == nil || node.Type != reader.NodeVector {
		return false
	}
	for index := 0; index < len(node.Children); index += 2 {
		if index < len(node.Children) && bindingSymbolsContainName(node.Children[index], name) {
			return true
		}
	}
	return false
}

func bindingSymbolsContainName(node *reader.RichNode, name string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSymbol {
		return node.Value == name
	}
	for _, child := range node.Children {
		if bindingSymbolsContainName(child, name) {
			return true
		}
	}
	return false
}

func isDirectTestAssertionComparison(node *reader.RichNode, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	if len(ancestors) == 0 {
		return false
	}
	parent := ancestors[len(ancestors)-1]
	if directChildIndex(parent, node) != 1 ||
		(!rules.CallResolvesTo(parent, "clojure.test/is") && !rules.CallResolvesTo(parent, "cljs.test/is")) {
		return false
	}
	for _, operator := range []string{"=", "==", "not=", ">", "<", ">=", "<=", "mod", "rem"} {
		if isCoreVerboseCall(node, operator) {
			return true
		}
	}
	return false
}

func isDefinitelyIntegral(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeNumber {
		// Decimal and scientific literals are numeric, but not provably integral
		// for the purposes of zero?/pos?/neg? rewrites.
		return !strings.ContainsAny(node.Value, ".eE")
	}
	hint := node.TypeHint
	switch hint {
	case "byte", "short", "int", "long", "Byte", "Short", "Integer", "Long",
		"java.lang.Byte", "java.lang.Short", "java.lang.Integer", "java.lang.Long":
		return true
	}
	if node.Type != reader.NodeList {
		return false
	}
	if isKnownIntegralInteropCall(node) {
		return true
	}
	for _, name := range []string{"int", "long", "unchecked-int", "unchecked-long", "count"} {
		if isCoreVerboseCall(node, name) && len(node.Children) == 2 {
			return true
		}
	}
	return isCoreVerboseCall(node, "compare") && len(node.Children) == 3
}

func isKnownIntegralInteropCall(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}
	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol || !strings.HasPrefix(head.Value, ".") {
		return false
	}
	receiver := node.Children[1]
	if receiver == nil {
		return false
	}
	method := strings.TrimPrefix(head.Value, ".")
	if (method == "length" || method == "size") && len(node.Children) != 2 {
		return false
	}
	if (method == "indexOf" || method == "lastIndexOf") && len(node.Children) != 3 {
		return false
	}
	if receiver.Type == reader.NodeString {
		return method == "indexOf" || method == "lastIndexOf" || method == "length"
	}
	switch receiver.Type {
	case reader.NodeVector:
		if method == "indexOf" || method == "lastIndexOf" || method == "size" {
			return true
		}
	case reader.NodeMap, reader.NodeSet:
		if method == "size" {
			return true
		}
	}
	if receiver.TypeHint == "" {
		return false
	}
	hint := strings.ToLower(strings.TrimSpace(receiver.TypeHint))
	switch method {
	case "indexOf", "lastIndexOf":
		return hint == "string" || hint == "java.lang.string" ||
			hint == "java.util.list" || hint == "java.util.collection"
	case "length":
		return hint == "string" || hint == "java.lang.string"
	case "size":
		return hint == "java.util.collection" || hint == "java.util.list" ||
			hint == "java.util.set" || hint == "java.util.map"
	default:
		return false
	}
}

func isInvalidKnownIntegralExpression(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList {
		return false
	}
	for _, name := range []string{"int", "long", "unchecked-int", "unchecked-long", "count"} {
		if isCoreVerboseCall(node, name) {
			return len(node.Children) != 2
		}
	}
	if isCoreVerboseCall(node, "compare") {
		return len(node.Children) != 3
	}
	if len(node.Children) < 2 || node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol ||
		!strings.HasPrefix(node.Children[0].Value, ".") || node.Children[1] == nil {
		return false
	}
	method := strings.TrimPrefix(node.Children[0].Value, ".")
	knownReceiver := node.Children[1].Type == reader.NodeString || node.Children[1].Type == reader.NodeVector ||
		node.Children[1].Type == reader.NodeMap || node.Children[1].Type == reader.NodeSet
	if !knownReceiver {
		return false
	}
	switch method {
	case "length", "size":
		return len(node.Children) != 2
	case "indexOf", "lastIndexOf":
		return len(node.Children) != 3
	default:
		return false
	}
}

type VerboseChecksRule struct {
	rules.Rule
	CheckNumericComparisons bool `json:"check_numeric_comparisons" yaml:"check_numeric_comparisons"`
	CheckBooleanComparisons bool `json:"check_boolean_comparisons" yaml:"check_boolean_comparisons"`
	CheckNilComparisons     bool `json:"check_nil_comparisons" yaml:"check_nil_comparisons"`
	CheckMathOperations     bool `json:"check_math_operations" yaml:"check_math_operations"`
}

func (r *VerboseChecksRule) Meta() rules.Rule {
	return r.Rule
}

var (
	numericComparisons map[string]map[string]string
	booleanComparisons map[string]string
	mathOperations     map[string]map[string]string
	verboseChecksOnce  sync.Once
)

func initVerboseChecksMaps() {
	verboseChecksOnce.Do(func() {
		numericComparisons = map[string]map[string]string{
			"=": {
				"0": "zero?",
			},
			">": {
				"0": "pos?",
			},
			"<": {
				"0": "neg?",
			},
			"<=": {
				"0": "not-pos",
			},
			">=": {
				"0": "not-neg",
			},
		}

		booleanComparisons = map[string]string{
			"true":  "true?",
			"false": "false?",
		}

		mathOperations = map[string]map[string]string{
			"+": {
				"1": "inc",
			},
			"-": {
				"1": "dec",
			},
		}
	})
}

func (r *VerboseChecksRule) detectNumericComparison(node *reader.RichNode, context map[string]interface{}) (*rules.Finding, bool) {
	initVerboseChecksMaps()

	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil, false
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol {
		return nil, false
	}

	operator := opNode.Value
	if !isCoreVerboseCall(node, operator) {
		return nil, false
	}
	if !hasProvenCallShape(node, context) {
		return nil, false
	}
	comparisons, exists := numericComparisons[operator]
	if !exists {
		return nil, false
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]
	var constantValue, variableExpr string
	var variableNode *reader.RichNode
	var suggestion string

	if arg1.Type == reader.NodeNumber {
		constantValue = arg1.Value
		variableExpr = getVerboseNodeText(arg2)
		variableNode = arg2
		if idiomaticFunc, exists := comparisons[constantValue]; exists {
			if operator == "=" {
				suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			} else if operator == ">" && constantValue == "0" {
				suggestion = fmt.Sprintf("(neg? %s)", variableExpr)
			} else if operator == "<" && constantValue == "0" {
				suggestion = fmt.Sprintf("(pos? %s)", variableExpr)
			} else if operator == "<=" && constantValue == "0" {
				suggestion = fmt.Sprintf("(>= %s 0)", variableExpr)
			} else if operator == ">=" && constantValue == "0" {
				suggestion = fmt.Sprintf("(<= %s 0)", variableExpr)
			}
		}
	} else if arg2.Type == reader.NodeNumber {
		constantValue = arg2.Value
		variableExpr = getVerboseNodeText(arg1)
		variableNode = arg1
		if idiomaticFunc, exists := comparisons[constantValue]; exists {
			if operator == "=" || operator == ">" || operator == "<" {
				suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			} else if operator == "<=" && constantValue == "0" {
				suggestion = fmt.Sprintf("(not (pos? %s))", variableExpr)
			} else if operator == ">=" && constantValue == "0" {
				suggestion = fmt.Sprintf("(not (neg? %s))", variableExpr)
			}
		}
	}

	// The replacements below are not equivalent for arbitrary Clojure values:
	// comparisons can return false for nil or heterogeneous values while
	// zero?/pos?/neg? throw. Require a statically evidenced numeric operand and
	// avoid reporting constant folding as a style smell.
	if suggestion != "" && variableNode != nil && variableNode.Type != reader.NodeNumber {
		if !isCoreReplacementAvailable(context, suggestedCoreName(suggestion)) {
			return nil, false
		}
		if rules.FactsForNode(variableNode, context).Constant {
			return nil, false
		}
		if isInvalidKnownIntegralExpression(variableNode) {
			return nil, false
		}
		originalExpr := fmt.Sprintf("(%s %s %s)", operator, getVerboseNodeText(arg1), getVerboseNodeText(arg2))
		return &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Verbose numeric comparison: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
			Filepath: "",
			Location: node.Location,
			Severity: r.Severity,
		}, isDefinitelyIntegral(variableNode)
	}

	return nil, false
}

func (r *VerboseChecksRule) detectBooleanComparison(node *reader.RichNode, context map[string]interface{}) *rules.Finding {
	initVerboseChecksMaps()

	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol || opNode.Value != "=" || !isCoreVerboseCall(node, "=") {
		return nil
	}
	if !hasProvenCallShape(node, context) {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	var constantValue, variableExpr string
	var suggestion string

	if arg1.Type == reader.NodeBool && (arg1.Value == "true" || arg1.Value == "false") {
		constantValue = arg1.Value
		variableExpr = getVerboseNodeText(arg2)
	} else if arg2.Type == reader.NodeBool && (arg2.Value == "true" || arg2.Value == "false") {
		constantValue = arg2.Value
		variableExpr = getVerboseNodeText(arg1)
	}

	if constantValue != "" {
		if idiomaticFunc, exists := booleanComparisons[constantValue]; exists {
			if !isCoreReplacementAvailable(context, idiomaticFunc) {
				return nil
			}
			suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			originalExpr := fmt.Sprintf("(%s %s %s)", opNode.Value, getVerboseNodeText(arg1), getVerboseNodeText(arg2))
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Verbose boolean comparison: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
				Filepath: "",
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

func (r *VerboseChecksRule) detectNilComparison(node *reader.RichNode, context map[string]interface{}) *rules.Finding {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol || (opNode.Value != "=" && opNode.Value != "not=") ||
		!isCoreVerboseCall(node, opNode.Value) {
		return nil
	}
	if !hasProvenCallShape(node, context) {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	var variableExpr string
	var isNilComparison bool

	if arg1.Type == reader.NodeNil {
		variableExpr = getVerboseNodeText(arg2)
		isNilComparison = true
	} else if arg2.Type == reader.NodeNil {
		variableExpr = getVerboseNodeText(arg1)
		isNilComparison = true
	}

	if isNilComparison {
		other := node.Children[2]
		if arg1.Type == reader.NodeNil {
			other = arg2
		} else {
			other = arg1
		}
		if rules.FactsForNode(other, context).Constant {
			return nil
		}
		var suggestion string
		if opNode.Value == "=" {
			suggestion = fmt.Sprintf("(nil? %s)", variableExpr)
		} else if opNode.Value == "not=" {
			suggestion = fmt.Sprintf("(some? %s)", variableExpr)
		} else {
			return nil
		}
		if !isCoreReplacementAvailable(context, suggestedCoreName(suggestion)) {
			return nil
		}
		originalExpr := fmt.Sprintf("(%s %s %s)", opNode.Value, getVerboseNodeText(arg1), getVerboseNodeText(arg2))
		return &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Verbose nil comparison: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
			Filepath: "",
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func (r *VerboseChecksRule) detectMathOperation(node *reader.RichNode, context map[string]interface{}) *rules.Finding {
	initVerboseChecksMaps()

	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol {
		return nil
	}

	operator := opNode.Value
	if !isCoreVerboseCall(node, operator) {
		return nil
	}
	if !hasProvenCallShape(node, context) {
		return nil
	}
	operations, exists := mathOperations[operator]
	if !exists {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	var constantValue, variableExpr string
	var suggestion string

	if operator == "+" {
		if arg1.Type == reader.NodeNumber && arg1.Value == "1" {
			constantValue = arg1.Value
			variableExpr = getVerboseNodeText(arg2)
		} else if arg2.Type == reader.NodeNumber && arg2.Value == "1" {
			constantValue = arg2.Value
			variableExpr = getVerboseNodeText(arg1)
		}
	} else if operator == "-" {

		if arg2.Type == reader.NodeNumber && arg2.Value == "1" {
			constantValue = arg2.Value
			variableExpr = getVerboseNodeText(arg1)
		}
	}

	if constantValue != "" {
		if idiomaticFunc, exists := operations[constantValue]; exists {
			if !isCoreReplacementAvailable(context, idiomaticFunc) {
				return nil
			}
			suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			originalExpr := fmt.Sprintf("(%s %s %s)", operator, getVerboseNodeText(arg1), getVerboseNodeText(arg2))
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Verbose math operation: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
				Filepath: "",
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

func (r *VerboseChecksRule) detectVerboseIf(node *reader.RichNode, context map[string]interface{}) *rules.Finding {
	if node.Type != reader.NodeList || len(node.Children) != 4 {
		return nil
	}
	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol || opNode.Value != "if" || !isCoreVerboseCall(node, "if") {
		return nil
	}
	if !hasProvenCallShape(node, context) {
		return nil
	}
	cond := node.Children[1]
	thenBranch := node.Children[2]
	elseBranch := node.Children[3]

	if thenBranch.Type == reader.NodeBool && elseBranch.Type == reader.NodeBool {
		if thenBranch.Value == "true" && elseBranch.Value == "false" {
			if !isCoreReplacementAvailable(context, "boolean") {
				return nil
			}
			suggestion := fmt.Sprintf("(boolean %s)", getVerboseNodeText(cond))
			originalExpr := fmt.Sprintf("(if %s true false)", getVerboseNodeText(cond))
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Verbose boolean if: `%s`. Consider using `%s`.", originalExpr, suggestion),
				Location: node.Location,
				Severity: r.Severity,
			}
		}
		if thenBranch.Value == "false" && elseBranch.Value == "true" {
			if !isCoreReplacementAvailable(context, "not") {
				return nil
			}
			suggestion := fmt.Sprintf("(not %s)", getVerboseNodeText(cond))
			originalExpr := fmt.Sprintf("(if %s false true)", getVerboseNodeText(cond))
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Verbose boolean if: `%s`. Consider using `%s`.", originalExpr, suggestion),
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}
	return nil
}

func (r *VerboseChecksRule) detectModComparison(node *reader.RichNode, context map[string]interface{}) (*rules.Finding, bool) {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil, false
	}
	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol || (opNode.Value != "=" && opNode.Value != "not=") ||
		!isCoreVerboseCall(node, opNode.Value) {
		return nil, false
	}
	if !hasProvenCallShape(node, context) {
		return nil, false
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	isMod := false
	var modArg string
	var modArgNode *reader.RichNode

	if arg1.Type == reader.NodeList && len(arg1.Children) == 3 &&
		(isCoreVerboseCall(arg1, "mod") || isCoreVerboseCall(arg1, "rem")) &&
		arg1.Children[2].Type == reader.NodeNumber && arg1.Children[2].Value == "2" {
		if arg2.Type == reader.NodeNumber && arg2.Value == "0" {
			isMod = true
			modArg = getVerboseNodeText(arg1.Children[1])
			modArgNode = arg1.Children[1]
		}
	} else if arg2.Type == reader.NodeList && len(arg2.Children) == 3 &&
		(isCoreVerboseCall(arg2, "mod") || isCoreVerboseCall(arg2, "rem")) &&
		arg2.Children[2].Type == reader.NodeNumber && arg2.Children[2].Value == "2" {
		if arg1.Type == reader.NodeNumber && arg1.Value == "0" {
			isMod = true
			modArg = getVerboseNodeText(arg2.Children[1])
			modArgNode = arg2.Children[1]
		}
	}

	if isMod {
		var suggestion string
		if opNode.Value == "=" {
			suggestion = fmt.Sprintf("(even? %s)", modArg)
		} else {
			suggestion = fmt.Sprintf("(odd? %s)", modArg)
		}
		if !isCoreReplacementAvailable(context, suggestedCoreName(suggestion)) {
			return nil, false
		}
		originalExpr := fmt.Sprintf("(%s %s %s)", opNode.Value, getVerboseNodeText(arg1), getVerboseNodeText(arg2))
		return &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Verbose parity check: `%s`. Consider using `%s`.", originalExpr, suggestion),
			Location: node.Location,
			Severity: r.Severity,
		}, isDefinitelyIntegral(modArgNode)
	}
	return nil, false
}

func getVerboseNodeText(node *reader.RichNode) string {
	if node == nil {
		return "nil"
	}

	switch node.Type {
	case reader.NodeSymbol, reader.NodeKeyword, reader.NodeString, reader.NodeNumber, reader.NodeBool, reader.NodeNil:
		return node.Value
	case reader.NodeList:
		if len(node.Children) > 0 {
			return "(" + getVerboseNodeText(node.Children[0]) + " ...)"
		}
		return "()"
	case reader.NodeVector:
		return "[...]"
	case reader.NodeMap:
		return "{...}"
	case reader.NodeSet:
		return "#{...}"
	default:
		return "..."
	}
}

func (r *VerboseChecksRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {

	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}
	if isDirectTestAssertionComparison(node, context) {
		return nil
	}

	if r.CheckNumericComparisons {
		if finding, proven := r.detectNumericComparison(node, context); finding != nil {
			finding.Filepath = filepath
			if proven {
				return finding
			}
			return rules.SetContextualFindingWithEvidence(finding, "The simplification depends on type, nil, overflow, and the compared value's return contract.", "numeric-type", "nil-behavior", "overflow", "return-contract")
		}
	}

	if r.CheckBooleanComparisons {
		if finding := r.detectBooleanComparison(node, context); finding != nil {
			finding.Filepath = filepath
			// Equality with the literal true/false is equivalent to true?/false?
			// for every Clojure value; no external contract is required.
			return finding
		}
	}

	if r.CheckNilComparisons {
		if finding := r.detectNilComparison(node, context); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	if r.CheckMathOperations {
		if finding := r.detectMathOperation(node, context); finding != nil {
			finding.Filepath = filepath
			return rules.SetContextualFindingWithEvidence(finding, "The arithmetic simplification may depend on numeric type and overflow.", "numeric-type", "overflow")
		}
	}

	if finding := r.detectVerboseIf(node, context); finding != nil {
		finding.Filepath = filepath
		// Both forms return a boolean and preserve Clojure truthiness exactly.
		return finding
	}

	if finding, proven := r.detectModComparison(node, context); finding != nil {
		finding.Filepath = filepath
		if proven {
			return finding
		}
		return rules.SetContextualFindingWithEvidence(finding, "The parity simplification depends on the numeric domain and input contract.", "numeric-domain", "input-contract")
	}

	return nil
}

func init() {
	defaultRule := &VerboseChecksRule{
		Rule: rules.Rule{
			ID:          "verbose-checks",
			Name:        "Verbose Checks",
			Description: "Detects verbose checks that can be simplified using idiomatic Clojure functions. This includes manual implementations of common checks like (= 0 x) instead of (zero? x), (= true x) instead of (true? x), (+ 1 x) instead of (inc x), and similar patterns. Based on idiomatic Clojure practices from bsless.github.io/code-smells.",
			Severity:    rules.SeverityHint,
		},
		CheckNumericComparisons: true,
		CheckBooleanComparisons: true,
		CheckNilComparisons:     true,
		CheckMathOperations:     true,
	}

	rules.RegisterRule(defaultRule)
}
