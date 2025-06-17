package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type PositionalReturnValuesRule struct {
	Rule
}

func (r *PositionalReturnValuesRule) Meta() Rule {
	return r.Rule
}

func isLiteralVector(node *reader.RichNode) bool {
	return node.Type == reader.NodeVector && len(node.Children) >= 2
}

func isLiteralList(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}

	firstChild := node.Children[0]
	return firstChild.Type == reader.NodeNumber ||
		firstChild.Type == reader.NodeString ||
		firstChild.Type == reader.NodeKeyword ||
		firstChild.Type == reader.NodeBool ||
		firstChild.Type == reader.NodeNil
}

func isPositionalCollection(node *reader.RichNode) bool {
	return (isLiteralVector(node) || isLiteralList(node)) && len(node.Children) >= 2
}

func (r *PositionalReturnValuesRule) findFunctionBody(node *reader.RichNode, fnType string) int {
	if fnType == "defn" {
		if len(node.Children) < 3 {
			return -1
		}
		bodyStartIndex := 2
		for i := 2; i < len(node.Children); i++ {
			if node.Children[i].Type == reader.NodeVector {
				return i + 1
			}
			if i == len(node.Children)-1 {
				return -1
			}
		}
		return bodyStartIndex
	} else if fnType == "fn" {
		if len(node.Children) < 2 {
			return -1
		}
		for i := 1; i < len(node.Children); i++ {
			if node.Children[i].Type == reader.NodeVector {
				return i + 1
			}
			if i == len(node.Children)-1 {
				return -1
			}
		}
	}
	return -1
}

func (r *PositionalReturnValuesRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || (firstChild.Value != "defn" && firstChild.Value != "fn") {
		return nil
	}

	bodyStartIndex := r.findFunctionBody(node, firstChild.Value)
	if bodyStartIndex == -1 || bodyStartIndex >= len(node.Children) {
		return nil
	}

	lastBodyForm := node.Children[len(node.Children)-1]

	if isPositionalCollection(lastBodyForm) {
		collectionType := "vector"
		if lastBodyForm.Type == reader.NodeList {
			collectionType = "list"
		}
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Function returns a literal %s with multiple values. Consider returning a map with descriptive keys instead of relying on positional values.", collectionType),
			Filepath: filepath,
			Location: lastBodyForm.Location,
			Severity: r.Severity,
		}
	}

	if lastBodyForm.Type == reader.NodeList && len(lastBodyForm.Children) > 0 &&
		lastBodyForm.Children[0].Type == reader.NodeSymbol && lastBodyForm.Children[0].Value == "let" {
		if len(lastBodyForm.Children) >= 3 {
			lastExprInLet := lastBodyForm.Children[len(lastBodyForm.Children)-1]
			if isPositionalCollection(lastExprInLet) {
				collectionType := "vector"
				if lastExprInLet.Type == reader.NodeList {
					collectionType = "list"
				}
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Function returns a literal %s with multiple values via a `let` form. Consider returning a map with descriptive keys instead of positional values.", collectionType),
					Filepath: filepath,
					Location: lastExprInLet.Location,
					Severity: r.Severity,
				}
			}
		}
	}

	return nil
}

func init() {
	RegisterRule(&PositionalReturnValuesRule{
		Rule: Rule{
			ID:          "positional-return-values",
			Name:        "Positional Return Values",
			Description: "Detects functions that return sequential collections (vectors or lists) where the meaning of the elements is implied by their position. It is recommended to return a map with descriptive keys.",
			Severity:    SeverityWarning,
		},
	})
}
