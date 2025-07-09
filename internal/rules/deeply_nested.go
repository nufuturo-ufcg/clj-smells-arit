package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type DeeplyNestedRule struct {
	Rule
	MaxDepth int `json:"max_depth" yaml:"max_depth"`
}

func (r *DeeplyNestedRule) Meta() Rule {
	return r.Rule
}

func calculateCallStackDepthRecursive(node *reader.RichNode) int {
	if node == nil {
		return 0
	}

	maxChildCallDepth := 0
	isCall := false

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		firstChildType := node.Children[0].Type

		if firstChildType == reader.NodeSymbol || firstChildType == reader.NodeKeyword {
			isCall = true
		}
	} else if node.Type == reader.NodeFnLiteral {

		isCall = false
		for _, child := range node.Children {
			depth := calculateCallStackDepthRecursive(child)
			if depth > maxChildCallDepth {
				maxChildCallDepth = depth
			}
		}
		return maxChildCallDepth
	}

	if isCall {

		for i := 1; i < len(node.Children); i++ {
			argDepth := calculateCallStackDepthRecursive(node.Children[i])
			if argDepth > maxChildCallDepth {
				maxChildCallDepth = argDepth
			}
		}
		return 1 + maxChildCallDepth
	} else if node.Type == reader.NodeList || node.Type == reader.NodeVector || node.Type == reader.NodeMap {

		for _, child := range node.Children {
			childDepth := calculateCallStackDepthRecursive(child)
			if childDepth > maxChildCallDepth {
				maxChildCallDepth = childDepth
			}
		}
		return maxChildCallDepth
	}

	return 0
}

func (r *DeeplyNestedRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type == reader.NodeList && len(node.Children) > 1 && node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || node.Children[0].Value == "defn-") {

		funcName := "anonymous-fn"
		if node.Children[1].Type == reader.NodeSymbol {
			funcName = node.Children[1].Value
		}

		maxOverallCallStackDepth := 0

		bodyStartIndex := 2
		if len(node.Children) > bodyStartIndex && node.Children[bodyStartIndex].Type == reader.NodeString {
			bodyStartIndex++
		}
		if len(node.Children) > bodyStartIndex && node.Children[bodyStartIndex].Type == reader.NodeMap {
			bodyStartIndex++
		}

		if bodyStartIndex < len(node.Children) {
			firstBodyNode := node.Children[bodyStartIndex]

			if firstBodyNode.Type == reader.NodeVector {

				for i := bodyStartIndex + 1; i < len(node.Children); i++ {
					exprNode := node.Children[i]
					depth := calculateCallStackDepthRecursive(exprNode)
					if depth > maxOverallCallStackDepth {
						maxOverallCallStackDepth = depth
					}
				}
			} else if firstBodyNode.Type == reader.NodeList {
				for _, arityForm := range firstBodyNode.Children {
					if arityForm.Type == reader.NodeList && len(arityForm.Children) > 1 && arityForm.Children[0].Type == reader.NodeVector {

						for i := 1; i < len(arityForm.Children); i++ {
							exprNode := arityForm.Children[i]
							depth := calculateCallStackDepthRecursive(exprNode)
							if depth > maxOverallCallStackDepth {
								maxOverallCallStackDepth = depth
							}
						}
					}
				}
			} else {
				for i := bodyStartIndex; i < len(node.Children); i++ {
					exprNode := node.Children[i]
					depth := calculateCallStackDepthRecursive(exprNode)
					if depth > maxOverallCallStackDepth {
						maxOverallCallStackDepth = depth
					}
				}
			}
		}

		if maxOverallCallStackDepth > r.MaxDepth {
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Function %q has a call stack depth of %d (max allowed %d). Consider breaking down deeply nested calls.", funcName, maxOverallCallStackDepth, r.MaxDepth),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}
	return nil
}

func init() {
	defaultRule := &DeeplyNestedRule{
		Rule: Rule{
			ID:          "deeply-nested",
			Name:        "Deeply Nested Call Stack",
			Description: "Detects deeply nested function call stacks within a function body. Deep call stacks can make code harder to read, debug, and may increase the risk of stack overflow.",
			Severity:    SeverityWarning,
		},
		MaxDepth: 15,
	}

	RegisterRule(defaultRule)
}
