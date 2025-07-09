package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type LongFunctionRule struct {
	Rule
	MaxLines int `json:"max_lines" yaml:"max_lines"`
}

func (r *LongFunctionRule) Meta() Rule {
	return r.Rule
}

func countActualLines(node *reader.RichNode) int {
	if node == nil {
		return 0
	}

	minLine, maxLine := findLineSpan(node)
	if minLine == 0 || maxLine == 0 {
		return 0
	}

	return maxLine - minLine + 1
}

func findLineSpan(node *reader.RichNode) (int, int) {
	if node == nil {
		return 0, 0
	}

	minLine := 0
	maxLine := 0

	if node.Location != nil {
		minLine = node.Location.StartLine
		maxLine = node.Location.StartLine
		if node.Location.EndLine > maxLine {
			maxLine = node.Location.EndLine
		}
	}

	for _, child := range node.Children {
		childMin, childMax := findLineSpan(child)
		if childMin > 0 {
			if minLine == 0 || childMin < minLine {
				minLine = childMin
			}
		}
		if childMax > 0 {
			if maxLine == 0 || childMax > maxLine {
				maxLine = childMax
			}
		}
	}

	return minLine, maxLine
}

func (r *LongFunctionRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *Finding {

	if node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "defn" {

		funcName := "unknown-function"
		if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
			funcName = node.Children[1].Value
		}

		actualLines := countActualLines(node)

		if actualLines > r.MaxLines {
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Function %q is too long: %d lines (max %d). Consider breaking it into smaller functions.", funcName, actualLines, r.MaxLines),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}
	return nil
}

func init() {
	defaultRule := &LongFunctionRule{
		Rule: Rule{
			ID:          "long-function",
			Name:        "Long Function",
			Description: "Functions should be kept short and focused. Long functions are harder to understand, test, and maintain.",
			Severity:    SeverityWarning,
		},
		MaxLines: 58,
	}

	RegisterRule(defaultRule)
}
