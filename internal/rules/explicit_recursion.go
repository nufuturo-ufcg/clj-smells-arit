package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type ExplicitRecursionRule struct {
	Rule
}

type RecursionPattern struct {
	Type        string
	Suggestion  string
	Confidence  string
	Description string
}

func (r *ExplicitRecursionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if !r.isLoopRecurForm(node) {
		return nil
	}

	pattern := r.analyzeRecursionPattern(node)
	if pattern == nil {
		return nil
	}

	severity := SeverityInfo
	if pattern.Confidence == "high" {
		severity = SeverityWarning
	}

	message := fmt.Sprintf("Explicit recursion detected (%s pattern). %s. Suggestion: %s",
		pattern.Type, pattern.Description, pattern.Suggestion)

	return &Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: severity,
	}
}

func (r *ExplicitRecursionRule) isLoopRecurForm(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || firstChild.Value != "loop" {
		return false
	}

	return r.hasRecurInBody(node)
}

func (r *ExplicitRecursionRule) hasRecurInBody(node *reader.RichNode) bool {
	for _, child := range node.Children {
		if r.containsRecur(child) {
			return true
		}
	}
	return false
}

func (r *ExplicitRecursionRule) containsRecur(node *reader.RichNode) bool {
	if node == nil {
		return false
	}

	if node.Type == reader.NodeSymbol && node.Value == "recur" {
		return true
	}

	for _, child := range node.Children {
		if r.containsRecur(child) {
			return true
		}
	}

	return false
}

func (r *ExplicitRecursionRule) analyzeRecursionPattern(node *reader.RichNode) *RecursionPattern {
	if len(node.Children) < 3 {
		return nil
	}

	bindings := node.Children[1]
	body := node.Children[2:]

	if pattern := r.detectAccumulatorPattern(bindings, body); pattern != nil {
		return pattern
	}

	if pattern := r.detectTransformationPattern(bindings, body); pattern != nil {
		return pattern
	}

	if pattern := r.detectFilteringPattern(bindings, body); pattern != nil {
		return pattern
	}

	if pattern := r.detectCountingPattern(bindings, body); pattern != nil {
		return pattern
	}

	if pattern := r.detectIterationPattern(bindings, body); pattern != nil {
		return pattern
	}

	return nil
}

func (r *ExplicitRecursionRule) detectAccumulatorPattern(bindings *reader.RichNode, body []*reader.RichNode) *RecursionPattern {
	if !r.hasAccumulatorBindings(bindings) {
		return nil
	}

	for _, bodyNode := range body {
		if r.hasAccumulatorUpdate(bodyNode) {
			confidence := "medium"
			if r.hasSimpleAccumulation(bodyNode) {
				confidence = "high"
			}

			return &RecursionPattern{
				Type:        "accumulator",
				Suggestion:  "Consider using `reduce` instead of explicit loop/recur",
				Confidence:  confidence,
				Description: "This loop accumulates values, which is typically better expressed with reduce",
			}
		}
	}

	return nil
}

func (r *ExplicitRecursionRule) detectTransformationPattern(bindings *reader.RichNode, body []*reader.RichNode) *RecursionPattern {
	if !r.hasCollectionBinding(bindings) {
		return nil
	}

	for _, bodyNode := range body {
		if r.hasElementTransformation(bodyNode) {
			return &RecursionPattern{
				Type:        "transformation",
				Suggestion:  "Consider using `map` or `mapv` instead of explicit loop/recur",
				Confidence:  "high",
				Description: "This loop transforms each element of a collection",
			}
		}
	}

	return nil
}

func (r *ExplicitRecursionRule) detectFilteringPattern(bindings *reader.RichNode, body []*reader.RichNode) *RecursionPattern {
	if !r.hasCollectionBinding(bindings) {
		return nil
	}

	for _, bodyNode := range body {
		if r.hasConditionalAccumulation(bodyNode) {
			return &RecursionPattern{
				Type:        "filtering",
				Suggestion:  "Consider using `filter` or `remove` instead of explicit loop/recur",
				Confidence:  "high",
				Description: "This loop filters elements based on a condition",
			}
		}
	}

	return nil
}

func (r *ExplicitRecursionRule) detectCountingPattern(bindings *reader.RichNode, body []*reader.RichNode) *RecursionPattern {
	if !r.hasCounterBinding(bindings) {
		return nil
	}

	for _, bodyNode := range body {
		if r.hasCounterIncrement(bodyNode) {
			return &RecursionPattern{
				Type:        "counting",
				Suggestion:  "Consider using `count`, `frequencies`, or `reduce` instead of explicit loop/recur",
				Confidence:  "medium",
				Description: "This loop counts elements or occurrences",
			}
		}
	}

	return nil
}

func (r *ExplicitRecursionRule) detectIterationPattern(bindings *reader.RichNode, body []*reader.RichNode) *RecursionPattern {
	if !r.hasSimpleIteration(bindings, body) {
		return nil
	}

	return &RecursionPattern{
		Type:        "iteration",
		Suggestion:  "Consider using `doseq`, `dotimes`, or `run!` instead of explicit loop/recur",
		Confidence:  "medium",
		Description: "This loop performs simple iteration without complex accumulation",
	}
}

func (r *ExplicitRecursionRule) hasAccumulatorBindings(bindings *reader.RichNode) bool {
	if bindings.Type != reader.NodeVector || len(bindings.Children) < 4 {
		return false
	}

	for i := 0; i < len(bindings.Children)-1; i += 2 {
		binding := bindings.Children[i]
		if binding.Type == reader.NodeSymbol {
			name := strings.ToLower(binding.Value)
			if strings.Contains(name, "acc") || strings.Contains(name, "result") ||
				strings.Contains(name, "sum") || strings.Contains(name, "total") {
				return true
			}
		}
	}

	return false
}

func (r *ExplicitRecursionRule) hasCollectionBinding(bindings *reader.RichNode) bool {
	if bindings.Type != reader.NodeVector || len(bindings.Children) < 2 {
		return false
	}

	for i := 0; i < len(bindings.Children)-1; i += 2 {
		binding := bindings.Children[i]
		if binding.Type == reader.NodeSymbol {
			name := strings.ToLower(binding.Value)
			if strings.Contains(name, "coll") || strings.Contains(name, "items") ||
				strings.Contains(name, "list") || strings.Contains(name, "seq") {
				return true
			}
		}
	}

	return false
}

func (r *ExplicitRecursionRule) hasCounterBinding(bindings *reader.RichNode) bool {
	if bindings.Type != reader.NodeVector || len(bindings.Children) < 2 {
		return false
	}

	for i := 0; i < len(bindings.Children)-1; i += 2 {
		binding := bindings.Children[i]
		if binding.Type == reader.NodeSymbol {
			name := strings.ToLower(binding.Value)
			if strings.Contains(name, "count") || strings.Contains(name, "cnt") ||
				strings.Contains(name, "n") || strings.Contains(name, "i") {
				return true
			}
		}
	}

	return false
}

func (r *ExplicitRecursionRule) hasAccumulatorUpdate(node *reader.RichNode) bool {
	return r.containsFunction(node, []string{"conj", "cons", "+", "*", "merge", "into", "assoc"})
}

func (r *ExplicitRecursionRule) hasSimpleAccumulation(node *reader.RichNode) bool {
	return r.containsFunction(node, []string{"+", "*", "conj", "cons"})
}

func (r *ExplicitRecursionRule) hasElementTransformation(node *reader.RichNode) bool {

	if node.Type == reader.NodeList && len(node.Children) >= 3 {
		if r.isFunction(node.Children[0], "conj") {

			if len(node.Children) > 2 {
				thirdArg := node.Children[2]
				return r.isTransformationCall(thirdArg)
			}
		}
	}

	for _, child := range node.Children {
		if r.hasElementTransformation(child) {
			return true
		}
	}

	return false
}

func (r *ExplicitRecursionRule) hasConditionalAccumulation(node *reader.RichNode) bool {
	if node.Type == reader.NodeList && len(node.Children) >= 2 {
		if r.isFunction(node.Children[0], "if") || r.isFunction(node.Children[0], "when") {

			for _, child := range node.Children[1:] {
				if r.containsFunction(child, []string{"conj", "cons"}) {
					return true
				}
			}
		}
	}

	for _, child := range node.Children {
		if r.hasConditionalAccumulation(child) {
			return true
		}
	}

	return false
}

func (r *ExplicitRecursionRule) hasCounterIncrement(node *reader.RichNode) bool {
	return r.containsFunction(node, []string{"inc", "+", "-"})
}

func (r *ExplicitRecursionRule) hasSimpleIteration(bindings *reader.RichNode, body []*reader.RichNode) bool {

	for _, bodyNode := range body {
		if r.hasAccumulatorUpdate(bodyNode) {
			return false
		}
	}

	for _, bodyNode := range body {
		if r.containsFunction(bodyNode, []string{"println", "print", "prn", "swap!", "reset!"}) {
			return true
		}
	}

	return false
}

func (r *ExplicitRecursionRule) containsFunction(node *reader.RichNode, functions []string) bool {
	if node == nil {
		return false
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		firstChild := node.Children[0]
		if firstChild.Type == reader.NodeSymbol {
			for _, fn := range functions {
				if firstChild.Value == fn {
					return true
				}
			}
		}
	}

	for _, child := range node.Children {
		if r.containsFunction(child, functions) {
			return true
		}
	}

	return false
}

func (r *ExplicitRecursionRule) isFunction(node *reader.RichNode, function string) bool {
	return node != nil && node.Type == reader.NodeSymbol && node.Value == function
}

func (r *ExplicitRecursionRule) isTransformationCall(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return false
	}

	transformFunctions := []string{
		"str", "keyword", "name", "inc", "dec", "+", "-", "*", "/",
		"upper-case", "lower-case", "trim", "reverse", "sort",
		"assoc", "dissoc", "update", "select-keys",
	}

	for _, fn := range transformFunctions {
		if firstChild.Value == fn {
			return true
		}
	}

	return false
}

func (r *ExplicitRecursionRule) Meta() Rule {
	return r.Rule
}

func init() {
	rule := &ExplicitRecursionRule{
		Rule: Rule{
			ID:          "explicit-recursion",
			Name:        "Explicit Recursion",
			Description: "Detects unnecessary explicit recursion (loop/recur) that could be replaced with higher-order functions like reduce, map, filter, etc. Explicit recursion should be used only when higher-order functions are insufficient.",
			Severity:    SeverityInfo,
		},
	}

	RegisterRule(rule)
}
