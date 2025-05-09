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

func countSignificantLines(node *reader.RichNode) int {
	if node == nil {
		return 0
	}

	count := 0
	if node.Type != reader.NodeComment && node.Type != reader.NodeNewline {
		count = 1

		if node.Type == reader.NodeList && len(node.Children) > 0 &&
			node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "let" {

			if len(node.Children) > 1 && node.Children[1].Type == reader.NodeVector {
				count += len(node.Children[1].Children) / 2
			}
		}
	}

	for _, child := range node.Children {
		count += countSignificantLines(child)
	}

	return count
}

func (r *LongFunctionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "defn" {

		funcName := "unknown-function"
		if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
			funcName = node.Children[1].Value
		}

		significantLines := countSignificantLines(node)

		if significantLines > r.MaxLines {
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Function %q is too long: %d significant lines (max %d). Consider breaking it into smaller functions. Each binding in let blocks counts as a significant line.", funcName, significantLines, r.MaxLines),
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
			Description: "Functions should be kept short and focused. Long functions are harder to understand, test, and maintain. Each binding in let blocks counts as a significant line.",
			Severity:    SeverityWarning,
		},
		MaxLines: 20,
	}

	RegisterRule(defaultRule)
}
