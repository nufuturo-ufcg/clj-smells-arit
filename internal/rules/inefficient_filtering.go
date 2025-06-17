package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type InefficientFilteringRule struct{}

func (r *InefficientFilteringRule) Meta() Rule {
	return Rule{
		ID:          "inefficient-filtering",
		Name:        "Inefficient Filtering",
		Description: "Detects inefficient filtering/removing patterns like `(first (filter ...))`, `(last (filter ...))`, `(first (remove ...))`, or `(last (remove ...))`. These often process the entire collection unnecessarily. Consider using `some`, reversing the collection, or other approaches depending on the specific case.",
		Severity:    SeverityHint,
	}
}

func (r *InefficientFilteringRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) != 2 {
		return nil
	}

	outerFuncNode := node.Children[0]
	innerCallNode := node.Children[1]

	if outerFuncNode.Type != reader.NodeSymbol || (outerFuncNode.Value != "first" && outerFuncNode.Value != "last") {
		return nil
	}

	if innerCallNode.Type != reader.NodeList || len(innerCallNode.Children) < 3 {
		return nil
	}

	innerFuncNode := innerCallNode.Children[0]

	if innerFuncNode.Type != reader.NodeSymbol || (innerFuncNode.Value != "filter" && innerFuncNode.Value != "remove") {
		return nil
	}

	outerFuncName := outerFuncNode.Value
	innerFuncName := innerFuncNode.Value
	pattern := fmt.Sprintf("(%s (%s ...))", outerFuncName, innerFuncName)

	var suggestion string
	if outerFuncName == "first" && innerFuncName == "filter" {
		suggestion = "Consider using `(some <predicate> <collection>)` which stops after the first match."
	} else if outerFuncName == "first" && innerFuncName == "remove" {
		suggestion = "Consider using `(some (complement <predicate>) <collection>)` or similar logic to find the first non-matching item directly."
	} else if outerFuncName == "last" {
		suggestion = "This processes the entire collection. If performance is critical, consider reversing the collection first (if applicable) or alternative approaches."
	} else {
		suggestion = "This pattern processes the entire collection unnecessarily."
	}

	meta := r.Meta()
	return &Finding{
		RuleID:   meta.ID,
		Message:  fmt.Sprintf("Using `%s` is often inefficient. %s", pattern, suggestion),
		Filepath: filepath,
		Location: node.Location,
		Severity: meta.Severity,
	}
}

func init() {
	RegisterRule(&InefficientFilteringRule{})
}
