package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type BlockingInsideGoRule struct {
	Rule
}

func (r *BlockingInsideGoRule) Meta() Rule {
	return r.Rule
}

// ok
func (r *BlockingInsideGoRule) checkBlockingFunction(symbol string) bool {

	if strings.Contains(symbol, "!!") {
		return true
	}

	return false
}

// ok
func (r *BlockingInsideGoRule) findGoBlock(symbol string) bool {

	if symbol == "go" {
		return true
	}

	if strings.HasSuffix(symbol, "/go") {
		return true
	}

	return false
}

func (r *BlockingInsideGoRule) findBlockingFunction(node []*reader.RichNode) bool {

	for _, child := range node {
		if child.Type == reader.NodeSymbol && r.checkBlockingFunction(child.Value) {
			return true
		}
		if r.findBlockingFunction(child.Children) {
			return true
		}
	}
	return false
}

func (r *BlockingInsideGoRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *Finding {

	if node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		r.findGoBlock(node.Children[0].Value) && r.findBlockingFunction(node.Children[1:]) {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Blocking function detected within the GO block %s.", node.Children[0].Value),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func init() {
	defaultRule := &BlockingInsideGoRule{
		Rule: Rule{
			ID:          "blocking-inside-go",
			Name:        "Blocking Inside GO",
			Description: "Using blocking functions like this within a GO block violates its non-blocking purpose.",
			Severity:    SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
}
