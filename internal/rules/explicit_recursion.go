package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type ExplicitRecursionRule struct {
	Rule
}

type RecursionPattern struct {
	Type        string
	Suggestion  string
	Description string
}

func (r *ExplicitRecursionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if !isSelfRecursiveFunc(node) {
		return nil
	}

	pattern := analyzeSelfRecursionPattern(node)
	if pattern == nil {
		return nil
	}

	functionName := getFunctionName(node)
	message := fmt.Sprintf("Explicit recursion detected in '%s' (%s). %s. Suggestion: %s",
		functionName, pattern.Type, pattern.Description, pattern.Suggestion)

	return &Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func isSelfRecursiveFunc(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return false
	}
	if node.Children[0].Value != "defn" && node.Children[0].Value != "defn-" {
		return false
	}
	funcName := getFunctionName(node)
	if funcName == "" {
		return false
	}

	for _, child := range node.Children[2:] {
		if containsRecursiveCall(child, funcName) {
			return true
		}
	}
	return false
}

func containsRecursiveCall(node *reader.RichNode, funcName string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == funcName {
		return true
	}
	for _, child := range node.Children {
		if containsRecursiveCall(child, funcName) {
			return true
		}
	}
	return false
}

func analyzeSelfRecursionPattern(node *reader.RichNode) *RecursionPattern {
	if len(node.Children) == 0 {
		return nil
	}
	body := node.Children[len(node.Children)-1]
	if body.Type != reader.NodeList || len(body.Children) < 4 || body.Children[0].Value != "if" {
		return nil
	}

	condNode := body.Children[1]
	elseNode := body.Children[3]

	if isFilterPattern(condNode, elseNode) {
		return &RecursionPattern{Type: "filtering pattern", Suggestion: "Consider using `filter`", Description: "This function filters a collection"}
	}

	if isMapPattern(condNode, elseNode) {
		return &RecursionPattern{Type: "transformation (map) pattern", Suggestion: "Consider using `map`", Description: "This function transforms a collection"}
	}

	if isReducePattern(condNode, elseNode) {
		return &RecursionPattern{Type: "accumulator (reduce) pattern", Suggestion: "Consider using `reduce`", Description: "This function accumulates a value from a collection"}
	}
	return nil
}

func isFilterPattern(cond, elseNode *reader.RichNode) bool {

	isBaseCase := cond.Type == reader.NodeList && len(cond.Children) > 0 && cond.Children[0].Value == "empty?"

	isDirectNestedIf := elseNode.Type == reader.NodeList && len(elseNode.Children) > 0 && elseNode.Children[0].Value == "if"

	isLetWithNestedIf := false
	if elseNode.Type == reader.NodeList && len(elseNode.Children) > 0 && elseNode.Children[0].Value == "let" {

		for i := 2; i < len(elseNode.Children); i++ {
			child := elseNode.Children[i]
			if child.Type == reader.NodeList && len(child.Children) > 0 && child.Children[0].Value == "if" {
				isLetWithNestedIf = true
				break
			}
		}
	}

	return isBaseCase && (isDirectNestedIf || isLetWithNestedIf)
}
func isMapPattern(cond, elseNode *reader.RichNode) bool {

	isBaseCase := cond.Type == reader.NodeList && len(cond.Children) > 0 && cond.Children[0].Value == "empty?"
	isCons := elseNode.Type == reader.NodeList && len(elseNode.Children) > 0 && elseNode.Children[0].Value == "cons"
	return isBaseCase && isCons
}
func isReducePattern(cond, elseNode *reader.RichNode) bool {

	isBaseCase := cond.Type == reader.NodeList && len(cond.Children) > 0 && cond.Children[0].Value == "empty?"
	isOperator := elseNode.Type == reader.NodeList && len(elseNode.Children) > 1 && elseNode.Children[0].Type == reader.NodeSymbol
	return isBaseCase && isOperator
}

func getFunctionName(node *reader.RichNode) string {
	if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
		return node.Children[1].Value
	}
	return ""
}

func (r *ExplicitRecursionRule) Meta() Rule {
	return r.Rule
}

func init() {
	rule := &ExplicitRecursionRule{
		Rule: Rule{
			ID:          "explicit-recursion",
			Name:        "Explicit Recursion",
			Description: "Detects unnecessary explicit recursion that could be replaced with higher-order functions like reduce, map, filter, etc. Explicit recursion should be used only when higher-order functions are insufficient.",
			Severity:    SeverityInfo,
		},
	}
	RegisterRule(rule)
}
