package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type VerboseChecksRule struct {
	Rule
	CheckNumericComparisons bool `json:"check_numeric_comparisons" yaml:"check_numeric_comparisons"`
	CheckBooleanComparisons bool `json:"check_boolean_comparisons" yaml:"check_boolean_comparisons"`
	CheckNilComparisons     bool `json:"check_nil_comparisons" yaml:"check_nil_comparisons"`
	CheckMathOperations     bool `json:"check_math_operations" yaml:"check_math_operations"`
}

func (r *VerboseChecksRule) Meta() Rule {
	return r.Rule
}

var numericComparisons = map[string]map[string]string{
	"=": {
		"0": "zero?",
	},
	">": {
		"0": "pos?",
	},
	"<": {
		"0": "neg?",
	},
}

var booleanComparisons = map[string]string{
	"true":  "true?",
	"false": "false?",
}

var mathOperations = map[string]map[string]string{
	"+": {
		"1": "inc",
	},
	"-": {
		"1": "dec",
	},
}

func (r *VerboseChecksRule) detectNumericComparison(node *reader.RichNode) *Finding {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol {
		return nil
	}

	operator := opNode.Value
	comparisons, exists := numericComparisons[operator]
	if !exists {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	var constantValue, variableExpr string
	var suggestion string

	if arg1.Type == reader.NodeNumber {
		constantValue = arg1.Value
		variableExpr = getVerboseNodeText(arg2)
		if idiomaticFunc, exists := comparisons[constantValue]; exists {
			if operator == "=" {
				suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			} else if operator == ">" && constantValue == "0" {

				suggestion = fmt.Sprintf("(neg? %s)", variableExpr)
			} else if operator == "<" && constantValue == "0" {

				suggestion = fmt.Sprintf("(pos? %s)", variableExpr)
			}
		}
	} else if arg2.Type == reader.NodeNumber {
		constantValue = arg2.Value
		variableExpr = getVerboseNodeText(arg1)
		if idiomaticFunc, exists := comparisons[constantValue]; exists {
			suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
		}
	}

	if suggestion != "" {
		originalExpr := fmt.Sprintf("(%s %s %s)", operator, getVerboseNodeText(arg1), getVerboseNodeText(arg2))
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Verbose numeric comparison: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
			Filepath: "",
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func (r *VerboseChecksRule) detectBooleanComparison(node *reader.RichNode) *Finding {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol || opNode.Value != "=" {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	var constantValue, variableExpr string
	var suggestion string

	if arg1.Type == reader.NodeSymbol && (arg1.Value == "true" || arg1.Value == "false") {
		constantValue = arg1.Value
		variableExpr = getVerboseNodeText(arg2)
	} else if arg2.Type == reader.NodeSymbol && (arg2.Value == "true" || arg2.Value == "false") {
		constantValue = arg2.Value
		variableExpr = getVerboseNodeText(arg1)
	}

	if constantValue != "" {
		if idiomaticFunc, exists := booleanComparisons[constantValue]; exists {
			suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			originalExpr := fmt.Sprintf("(= %s %s)", getVerboseNodeText(arg1), getVerboseNodeText(arg2))
			return &Finding{
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

func (r *VerboseChecksRule) detectNilComparison(node *reader.RichNode) *Finding {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol || opNode.Value != "=" {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	var variableExpr string
	var isNilComparison bool

	if arg1.Type == reader.NodeSymbol && arg1.Value == "nil" {
		variableExpr = getVerboseNodeText(arg2)
		isNilComparison = true
	} else if arg2.Type == reader.NodeSymbol && arg2.Value == "nil" {
		variableExpr = getVerboseNodeText(arg1)
		isNilComparison = true
	}

	if isNilComparison {
		suggestion := fmt.Sprintf("(nil? %s)", variableExpr)
		originalExpr := fmt.Sprintf("(= %s %s)", getVerboseNodeText(arg1), getVerboseNodeText(arg2))
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Verbose nil comparison: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
			Filepath: "",
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func (r *VerboseChecksRule) detectMathOperation(node *reader.RichNode) *Finding {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol {
		return nil
	}

	operator := opNode.Value
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
			suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			originalExpr := fmt.Sprintf("(%s %s %s)", operator, getVerboseNodeText(arg1), getVerboseNodeText(arg2))
			return &Finding{
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

func getVerboseNodeText(node *reader.RichNode) string {
	if node == nil {
		return "nil"
	}

	switch node.Type {
	case reader.NodeSymbol, reader.NodeKeyword, reader.NodeString, reader.NodeNumber:
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

func (r *VerboseChecksRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	if r.CheckNumericComparisons {
		if finding := r.detectNumericComparison(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	if r.CheckBooleanComparisons {
		if finding := r.detectBooleanComparison(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	if r.CheckNilComparisons {
		if finding := r.detectNilComparison(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	if r.CheckMathOperations {
		if finding := r.detectMathOperation(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	return nil
}

func init() {
	defaultRule := &VerboseChecksRule{
		Rule: Rule{
			ID:          "verbose-checks",
			Name:        "Verbose Checks",
			Description: "Detects verbose checks that can be simplified using idiomatic Clojure functions. This includes manual implementations of common checks like (= 0 x) instead of (zero? x), (= true x) instead of (true? x), (+ 1 x) instead of (inc x), and similar patterns. Based on idiomatic Clojure practices from bsless.github.io/code-smells.",
			Severity:    SeverityHint,
		},
		CheckNumericComparisons: true,
		CheckBooleanComparisons: true,
		CheckNilComparisons:     true,
		CheckMathOperations:     true,
	}

	RegisterRule(defaultRule)
}
