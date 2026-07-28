package traditional

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type LongParameterListRule struct {
	rules.Rule
	MaxParameters int `json:"max_parameters" yaml:"max_parameters"`
}

func (r *LongParameterListRule) Meta() rules.Rule {
	return r.Rule
}

func (r *LongParameterListRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {

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
			paramCount := 0
			optionalParamCount := 0
			variadic := false

			for _, param := range argsNode.Children {
				if param.Type == reader.NodeSymbol {
					if param.Value == "&" {
						variadic = true
						continue
					}

					if param.Value == "_" ||
						strings.HasPrefix(param.Value, ".") ||
						strings.Contains(param.Value, "/") {
						continue
					}

					if variadic {
						optionalParamCount++
					} else {
						paramCount++
					}
				} else if variadic && param.Type == reader.NodeVector {

					for _, optParam := range param.Children {
						if optParam.Type == reader.NodeSymbol &&
							optParam.Value != "_" &&
							!strings.HasPrefix(optParam.Value, ".") &&
							!strings.Contains(optParam.Value, "/") {
							optionalParamCount++
						}
					}
				}
			}

			if (paramCount + optionalParamCount) > r.MaxParameters {
				return &rules.Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Function %q has too many parameters: %d (max %d). Optional parameters: %d.", funcName, paramCount+optionalParamCount, r.MaxParameters, optionalParamCount),
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
		Rule: rules.Rule{
			ID:          "long-parameter-list",
			Name:        "Long Parameter List",
			Description: "Functions should not have an excessive number of parameters. Consider grouping related parameters into a map or record.",
			Severity:    rules.SeverityWarning,
		},
		MaxParameters: 9,
	}

	rules.RegisterRule(defaultRule)
}