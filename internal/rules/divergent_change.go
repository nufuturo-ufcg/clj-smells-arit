package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
)

type DivergentChangeRule struct {
	Rule
}

func NewDivergentChangeRule(cfg *config.Config) *DivergentChangeRule {
	return &DivergentChangeRule{
		Rule: Rule{
			ID:          "divergent-change",
			Name:        "Divergent Change",
			Description: "Detects functions that handle multiple unrelated responsibilities, leading to changes for different reasons.",
			Severity:    SeverityWarning,
		},
	}
}

func (r *DivergentChangeRule) Meta() Rule {
	return r.Rule
}

func (r *DivergentChangeRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || firstChild.Value != "defn" {
		return nil
	}

	if len(node.Children) < 3 {
		return nil
	}

	var hasIO bool
	var dataManipulationOperations int

	var visitBody func(n *reader.RichNode)
	visitBody = func(n *reader.RichNode) {
		if n == nil {
			return
		}

		if n.Type == reader.NodeList && len(n.Children) > 0 && n.Children[0].Type == reader.NodeSymbol {
			callName := n.Children[0].Value

			if isIOCall(callName) {
				hasIO = true
			} else if isDataTransformationCall(callName) {
				dataManipulationOperations++
			} else if isControlFlow(callName) {

				for i := 1; i < len(n.Children); i++ {
					visitBody(n.Children[i])
				}
				return
			}
		}

		for _, child := range n.Children {
			visitBody(child)
		}
	}

	bodyStartIndex := 3
	if len(node.Children) > bodyStartIndex && node.Children[bodyStartIndex].Type == reader.NodeString {
		bodyStartIndex++
	}
	if len(node.Children) > bodyStartIndex && node.Children[bodyStartIndex].Type == reader.NodeMap {
		bodyStartIndex++
	}

	for i := bodyStartIndex; i < len(node.Children); i++ {
		visitBody(node.Children[i])
	}

	if hasIO && dataManipulationOperations > 2 {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Function '%s' seems to handle multiple responsibilities (I/O and significant data manipulation). Consider separating these concerns.", getNodeName(node)),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func getNodeName(defnNode *reader.RichNode) string {
	if defnNode.Type == reader.NodeList && len(defnNode.Children) > 1 && defnNode.Children[0].Type == reader.NodeSymbol && defnNode.Children[0].Value == "defn" {
		if defnNode.Children[1].Type == reader.NodeSymbol {
			return defnNode.Children[1].Value
		}
	}
	return "<unknown_function>"
}

func isIOCall(funcName string) bool {
	ioFunctions := map[string]bool{
		"println": true,
		"print":   true,
		"slurp":   true,
		"spit":    true,
	}
	return ioFunctions[funcName]
}

func isDataTransformationCall(funcName string) bool {
	transformFunctions := map[string]bool{
		"str":    true,
		"assoc":  true,
		"dissoc": true,
		"get":    true,
		"get-in": true,
		"update": true,
		"conj":   true,
		"into":   true,
	}

	if funcName == "if" || funcName == "let" || funcName == "loop" || funcName == "cond" || funcName == "case" {
		return false
	}

	if !isIOCall(funcName) && !isControlFlow(funcName) && !strings.HasPrefix(funcName, ".") {

		return transformFunctions[funcName]
	}
	return false
}

func isControlFlow(funcName string) bool {
	controlFlowSymbols := map[string]bool{
		"if":      true,
		"when":    true,
		"cond":    true,
		"case":    true,
		"let":     true,
		"loop":    true,
		"binding": true,
		"doseq":   true,
		"dotimes": true,
		"for":     true,
	}
	return controlFlowSymbols[funcName]
}

func init() {
	RegisterRule(NewDivergentChangeRule(nil))
}
