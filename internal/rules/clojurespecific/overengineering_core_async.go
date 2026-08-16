package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type OverengineeringCoreAsyncRule struct{ rules.Rule }

func (r *OverengineeringCoreAsyncRule) Meta() rules.Rule { return r.Rule }

func coreAsyncOperation(node *reader.RichNode) string {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 ||
		node.Children[0].Type != reader.NodeSymbol {
		return ""
	}
	return node.Children[0].Value
}

func coreAsyncHasSuffix(symbol string, names ...string) bool {
	for _, name := range names {
		if symbol == name || strings.HasSuffix(symbol, "/"+name) {
			return true
		}
	}
	return false
}

func coreAsyncCountSingleValuePuts(node *reader.RichNode, channel string) (int, bool) {
	if node == nil {
		return 0, false
	}
	op := coreAsyncOperation(node)
	if coreAsyncHasSuffix(op, "loop", "go-loop", "doseq", "pipeline", "pipeline-blocking", "pipeline-async", "mult", "pub") {
		return 0, true
	}
	count := 0
	if coreAsyncHasSuffix(op, ">!", ">!!", "put!") && len(node.Children) >= 3 &&
		node.Children[1].Type == reader.NodeSymbol && node.Children[1].Value == channel {
		count++
	}
	for _, child := range node.Children {
		childCount, complex := coreAsyncCountSingleValuePuts(child, channel)
		count += childCount
		if complex {
			return count, true
		}
	}
	return count, false
}

func coreAsyncSingleValueChannel(node *reader.RichNode) (*reader.RichNode, string) {
	if coreAsyncOperation(node) != "let" || len(node.Children) < 4 || node.Children[1].Type != reader.NodeVector {
		return nil, ""
	}
	bindings := node.Children[1]
	for index := 0; index+1 < len(bindings.Children); index += 2 {
		name, value := bindings.Children[index], bindings.Children[index+1]
		if name.Type != reader.NodeSymbol || !coreAsyncHasSuffix(coreAsyncOperation(value), "chan") {
			continue
		}
		last := node.Children[len(node.Children)-1]
		if last.Type != reader.NodeSymbol || last.Value != name.Value {
			continue
		}
		puts, complex := coreAsyncCountSingleValuePuts(node, name.Value)
		if puts != 1 || complex {
			continue
		}
		return value, name.Value
	}
	return nil, ""
}

func (r *OverengineeringCoreAsyncRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *rules.Finding {
	value, channel := coreAsyncSingleValueChannel(node)
	if value != nil {
		return &rules.Finding{
			RuleID: r.ID, Filepath: filepath, Location: value.Location, Severity: r.Severity,
			Message: fmt.Sprintf("Channel %q is used only to return one value; prefer a direct value, future, or promise.", channel),
		}
	}
	return nil
}

func init() {
	rules.RegisterRule(&OverengineeringCoreAsyncRule{Rule: rules.Rule{
		ID: "overengineering-with-core-async", Name: "Overengineering with core.async",
		Description: "Detects a channel allocated, written once, and directly returned as a single-value result.",
		Severity:    rules.SeverityHint,
	}})
}
