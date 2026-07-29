package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type BlockingInsideGoRule struct {
	rules.Rule
}

func (r *BlockingInsideGoRule) Meta() rules.Rule {
	return r.Rule
}

func (r *BlockingInsideGoRule) checkBlockingFunction(symbol string) bool {
	if strings.Contains(symbol, "!!") {
		return true
	}
	
	switch symbol {
	case "Thread/sleep", "slurp", "spit", "await", "deref", "future-call", "locking", "@":
		return true
	}
	
	if strings.HasPrefix(symbol, "clj-http.client/") || strings.HasPrefix(symbol, "http/") {
		return true
	}
	
	if strings.Contains(symbol, "jdbc/execute!") {
		return true
	}
	
	if symbol == ".readLine" || symbol == ".acquire" || symbol == "java.net.Socket." {
		return true
	}

	return false
}

// ok
func (r *BlockingInsideGoRule) findGoBlock(symbol string) bool {

	if symbol == "go" || symbol == "go-loop" {
		return true
	}

	if strings.HasSuffix(symbol, "/go") || strings.HasSuffix(symbol, "/go-loop") {
		return true
	}

	return false
}

func (r *BlockingInsideGoRule) findBlockingFunction(node []*reader.RichNode, visited map[*reader.RichNode]bool) bool {
	if visited == nil {
		visited = make(map[*reader.RichNode]bool)
	}
	
	for _, child := range node {
		if child == nil {
			continue
		}
		if visited[child] {
			continue
		}
		visited[child] = true

		if child.Type == reader.NodeSymbol {
			if r.checkBlockingFunction(child.Value) {
				return true
			}
			
			if child.ResolvedDefinition != nil {
				if r.findBlockingFunction([]*reader.RichNode{child.ResolvedDefinition}, visited) {
					return true
				}
			}
		}
		
		if r.findBlockingFunction(child.Children, visited) {
			return true
		}
	}
	return false
}

func (r *BlockingInsideGoRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *rules.Finding {

	if node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		r.findGoBlock(node.Children[0].Value) && r.findBlockingFunction(node.Children[1:], nil) {
		return &rules.Finding{
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
		Rule: rules.Rule{
			ID:          "blocking-inside-go",
			Name:        "Blocking Inside GO",
			Description: "Using blocking functions like this within a GO block violates its non-blocking purpose.",
			Severity:    rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}