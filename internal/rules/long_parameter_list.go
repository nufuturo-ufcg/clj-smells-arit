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

	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		fnType := node.Children[0].Value
		if fnType != "defn" && fnType != "defn-" && fnType != "defmacro" {
			return nil
		}

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

			fixedParamCount := reader.CountFunctionParameters(argsNode)

			optionalParamCount := r.countOptionalParameters(argsNode)

			totalConceptualParams := fixedParamCount + optionalParamCount

			if totalConceptualParams > r.MaxParameters*2 {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Function %q has excessive parameter complexity: %d required + %d optional = %d total concepts. Consider refactoring into smaller, focused functions.", funcName, fixedParamCount, optionalParamCount, totalConceptualParams),
					Filepath: filepath,
					Location: argsNode.Location,
					Severity: SeverityWarning,
				}
			}

			if fixedParamCount > r.MaxParameters {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Function %q has too many required parameters: %d (max %d). Consider using a map or breaking into smaller functions.", funcName, fixedParamCount, r.MaxParameters),
					Filepath: filepath,
					Location: argsNode.Location,
					Severity: r.Severity,
				}
			}

			if optionalParamCount > r.MaxParameters {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Function %q has too many optional parameters: %d (max %d). Consider using a configuration map or splitting functionality.", funcName, optionalParamCount, r.MaxParameters),
					Filepath: filepath,
					Location: argsNode.Location,
					Severity: SeverityInfo,
				}
			}
		}
	}
	return nil
}

func (r *LongParameterListRule) countOptionalParameters(paramsNode *reader.RichNode) int {
	if paramsNode == nil || paramsNode.Type != reader.NodeVector {
		return 0
	}

	foundVariadic := false
	optionalCount := 0

	for _, param := range paramsNode.Children {
		if param.Type == reader.NodeSymbol && param.Value == "&" {
			foundVariadic = true
			continue
		}

		if foundVariadic {

			if param.Type == reader.NodeMap {
				optionalCount += r.countKeywordMapParameters(param)
			} else {

				optionalCount = 1
			}
		}
	}

	return optionalCount
}

func (r *LongParameterListRule) countKeywordMapParameters(mapNode *reader.RichNode) int {
	if mapNode == nil || mapNode.Type != reader.NodeMap {
		return 0
	}

	totalOptionalParams := 0

	for i := 0; i < len(mapNode.Children); i += 2 {
		if i+1 >= len(mapNode.Children) {
			break
		}

		keyNode := mapNode.Children[i]
		valueNode := mapNode.Children[i+1]

		if keyNode.Type == reader.NodeKeyword {

			switch keyNode.Value {
			case ":keys", ":strs", ":syms":
				if valueNode.Type == reader.NodeVector {
					totalOptionalParams += len(valueNode.Children)
				}
			case ":or":

				continue
			case ":as":

				continue
			default:

				totalOptionalParams++
			}
		}
	}

	return totalOptionalParams
}

func init() {

	defaultRule := &LongParameterListRule{
		Rule: Rule{
			ID:          "long-parameter-list",
			Name:        "Long Parameter List",
			Description: "Functions should not have an excessive number of parameters. Consider grouping related parameters into a map or record.",
			Severity:    SeverityWarning,
		},
		MaxParameters: 9,
	}

	RegisterRule(defaultRule)
}
