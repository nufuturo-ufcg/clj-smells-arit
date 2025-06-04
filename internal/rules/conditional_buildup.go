package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type ConditionalBuildupRule struct {
	Rule
	MaxNestingDepth    int  `json:"max_nesting_depth" yaml:"max_nesting_depth"`
	MinConditions      int  `json:"min_conditions" yaml:"min_conditions"`
	CheckEqualityChain bool `json:"check_equality_chain" yaml:"check_equality_chain"`
}

func (r *ConditionalBuildupRule) Meta() Rule {
	return r.Rule
}

var conditionalForms = map[string]bool{
	"if":       true,
	"if-not":   true,
	"when":     true,
	"when-not": true,
	"if-let":   true,
	"when-let": true,
}

func isConditionalForm(symbol string) bool {
	return conditionalForms[symbol]
}

func (r *ConditionalBuildupRule) countNestedConditionals(node *reader.RichNode, depth int) (int, []string) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return depth, nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return depth, nil
	}

	var conditions []string
	maxDepth := depth

	if isConditionalForm(firstChild.Value) {
		conditions = append(conditions, firstChild.Value)
		currentDepth := depth + 1

		for _, child := range node.Children[1:] {
			childDepth, childConditions := r.countNestedConditionals(child, currentDepth)
			if childDepth > maxDepth {
				maxDepth = childDepth
			}
			conditions = append(conditions, childConditions...)
		}
	} else {

		for _, child := range node.Children {
			childDepth, childConditions := r.countNestedConditionals(child, depth)
			if childDepth > maxDepth {
				maxDepth = childDepth
			}
			conditions = append(conditions, childConditions...)
		}
	}

	return maxDepth, conditions
}

func (r *ConditionalBuildupRule) detectEqualityChain(node *reader.RichNode) *Finding {
	if !r.CheckEqualityChain {
		return nil
	}

	equalityChain := r.findEqualityChain(node, "", 0)
	if len(equalityChain) >= 3 {
		meta := r.Meta()
		return &Finding{
			RuleID: meta.ID,
			Message: fmt.Sprintf("Equality chain detected (%d comparisons). Consider using 'case' for better readability: (case %s %s)",
				len(equalityChain), equalityChain[0], strings.Join(equalityChain[1:], " ")),
			Filepath: "",
			Location: node.Location,
			Severity: meta.Severity,
		}
	}

	return nil
}

func (r *ConditionalBuildupRule) findEqualityChain(node *reader.RichNode, variable string, count int) []string {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || firstChild.Value != "if" {
		return nil
	}

	condition := node.Children[1]
	if condition.Type == reader.NodeList && len(condition.Children) == 3 {
		conditionFunc := condition.Children[0]
		if conditionFunc.Type == reader.NodeSymbol && conditionFunc.Value == "=" {
			var1 := getNodeTextForConditional(condition.Children[1])
			var2 := getNodeTextForConditional(condition.Children[2])

			if variable == "" {
				variable = var1
			}

			if var1 == variable {
				chain := []string{variable, var2}

				if len(node.Children) > 3 {
					elseClause := node.Children[3]
					nestedChain := r.findEqualityChain(elseClause, variable, count+1)
					if nestedChain != nil {
						chain = append(chain, nestedChain[1:]...)
					}
				}

				return chain
			}
		}
	}

	return nil
}

func (r *ConditionalBuildupRule) detectNestedConditionals(node *reader.RichNode) *Finding {
	depth, conditions := r.countNestedConditionals(node, 0)

	if depth > r.MaxNestingDepth {
		meta := r.Meta()
		conditionTypes := make(map[string]int)
		for _, cond := range conditions {
			conditionTypes[cond]++
		}

		var suggestion string
		if len(conditionTypes) == 1 && conditionTypes["if"] > 0 {
			suggestion = "Consider using 'cond' for multiple conditions"
		} else if len(conditionTypes) <= 2 {
			suggestion = "Consider using 'cond' to flatten the conditional structure"
		} else {
			suggestion = "Consider refactoring into smaller functions or using 'cond'"
		}

		return &Finding{
			RuleID: meta.ID,
			Message: fmt.Sprintf("Excessive conditional nesting detected (depth: %d, conditions: %d). %s. Nested conditionals make code harder to read and maintain.",
				depth, len(conditions), suggestion),
			Filepath: "",
			Location: node.Location,
			Severity: meta.Severity,
		}
	}

	return nil
}

func (r *ConditionalBuildupRule) detectMultipleIfElseChain(node *reader.RichNode) *Finding {
	ifCount := r.countIfElseChain(node)

	if ifCount >= r.MinConditions {
		meta := r.Meta()
		return &Finding{
			RuleID: meta.ID,
			Message: fmt.Sprintf("Multiple if-else chain detected (%d conditions). Consider using 'cond' for better readability: (cond condition1 result1 condition2 result2 :else default)",
				ifCount),
			Filepath: "",
			Location: node.Location,
			Severity: meta.Severity,
		}
	}

	return nil
}

func (r *ConditionalBuildupRule) countIfElseChain(node *reader.RichNode) int {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 3 {
		return 0
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || firstChild.Value != "if" {
		return 0
	}

	count := 1

	if len(node.Children) > 3 {
		elseClause := node.Children[3]
		count += r.countIfElseChain(elseClause)
	}

	return count
}

func getNodeTextForConditional(node *reader.RichNode) string {
	if node == nil {
		return "nil"
	}

	switch node.Type {
	case reader.NodeSymbol, reader.NodeKeyword, reader.NodeString, reader.NodeNumber:
		return node.Value
	case reader.NodeList:
		if len(node.Children) > 0 {
			return "(" + getNodeTextForConditional(node.Children[0]) + " ...)"
		}
		return "()"
	default:
		return "..."
	}
}

func (r *ConditionalBuildupRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || !isConditionalForm(firstChild.Value) {
		return nil
	}

	if finding := r.detectNestedConditionals(node); finding != nil {
		finding.Filepath = filepath
		return finding
	}

	if finding := r.detectMultipleIfElseChain(node); finding != nil {
		finding.Filepath = filepath
		return finding
	}

	if finding := r.detectEqualityChain(node); finding != nil {
		finding.Filepath = filepath
		return finding
	}

	return nil
}

func init() {
	defaultRule := &ConditionalBuildupRule{
		Rule: Rule{
			ID:          "conditional-buildup",
			Name:        "Conditional Build-Up",
			Description: "Detects excessive conditional construction that could be simplified using 'cond', 'case', or other more idiomatic structures. This includes deeply nested if/when statements, multiple if-else chains, and equality chains that would benefit from case statements. Based on idiomatic Clojure practices from bsless.github.io/code-smells.",
			Severity:    SeverityHint,
		},
		MaxNestingDepth:    3,
		MinConditions:      3,
		CheckEqualityChain: true,
	}

	RegisterRule(defaultRule)
}
