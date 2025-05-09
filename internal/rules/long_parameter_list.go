package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type LongParameterListRule struct {
	Rule
	MaxParameters int `json:"max_parameters" yaml:"max_parameters"`
}

func (r *LongParameterListRule) Meta() Rule {
	return r.Rule
}

func (r *LongParameterListRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "defn" {
		funcName := "unknown-function"
		if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
			funcName = node.Children[1].Value
		}

		var argsNode *reader.RichNode
		argsNodeIndex := 2
		if len(node.Children) > argsNodeIndex && node.Children[argsNodeIndex].Type == reader.NodeString {
			argsNodeIndex = 3
		}

		if len(node.Children) > argsNodeIndex {
			argsNode = node.Children[argsNodeIndex]
		} else {
			return nil
		}

		if argsNode != nil && argsNode.Type == reader.NodeVector {
			paramCount := len(argsNode.Children)
			if paramCount > r.MaxParameters {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Function %q has too many parameters: %d (max %d). Consider using a map.", funcName, paramCount, r.MaxParameters),
					Filepath: filepath,
					Location: argsNode.Location,
					Severity: r.Severity,
				}
			}
		} else {

		}
	}
	return nil
}

func init() {

	defaultRule := &LongParameterListRule{
		Rule: Rule{
			ID:          "long-parameter-list",
			Name:        "Long Parameter List",
			Description: "Functions should not have an excessive number of parameters. Consider grouping related parameters into a map or record.",
			Severity:    SeverityWarning,
		},
		MaxParameters: 5,
	}

	RegisterRule(defaultRule)
}
