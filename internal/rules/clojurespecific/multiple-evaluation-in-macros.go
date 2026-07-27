package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type MultipleEvaluationInMacrosRule struct {
	rules.Rule
}

func (r *MultipleEvaluationInMacrosRule) Meta() rules.Rule {
	return r.Rule
}

func (r *MultipleEvaluationInMacrosRule) extractParameters(paramsNode *reader.RichNode) map[string]int {
	parameters := make(map[string]int)

	if paramsNode == nil || paramsNode.Type != reader.NodeVector {
		return parameters
	}

	for _, currentParam := range paramsNode.Children {
		if currentParam != nil && currentParam.Type == reader.NodeSymbol {
			if currentParam.Value == "_" ||
				strings.HasPrefix(currentParam.Value, ".") ||
				strings.Contains(currentParam.Value, "/") || currentParam.Value == "&" {
				continue
			}
			parameters[currentParam.Value] = 0
		}
	}
	return parameters
}

func (r *MultipleEvaluationInMacrosRule) countParametersUses(node *reader.RichNode, count map[string]int) map[string]int {
	if node == nil {
		return count
	}

	if node.Type == reader.NodeSymbol {
		currentSymbol := node.Value
		if !strings.HasSuffix(currentSymbol, "#") {
			if strings.HasPrefix(currentSymbol, "~") {
				currentSymbol = strings.ReplaceAll(currentSymbol, "~", "")
			}
			if _, ok := count[currentSymbol]; ok {
				count[currentSymbol]++
			}
		}
	}

	for _, child := range node.Children {
		r.countParametersUses(child, count)
	}

	return count
}

func (r *MultipleEvaluationInMacrosRule) multipleEvaluation(node *reader.RichNode) string {
	if node == nil || len(node.Children) < 3 {
		return ""
	}

	// Skip macro name (index 1), docstring (optional), attr-map (optional)
	idx := 2
	for idx < len(node.Children) {
		child := node.Children[idx]
		if child != nil && (child.Type == reader.NodeString || child.Type == reader.NodeMap) {
			idx++
		} else {
			break
		}
	}

	if idx >= len(node.Children) {
		return ""
	}

	paramsNode := node.Children[idx]
	if paramsNode == nil {
		return ""
	}

	var bodyNodes []*reader.RichNode
	var paramsMap map[string]int

	if paramsNode.Type == reader.NodeVector {
		paramsMap = r.extractParameters(paramsNode)
		bodyNodes = node.Children[idx+1:]
	} else if paramsNode.Type == reader.NodeList {
		// Multi-arity macro: (defmacro name ([p1] body1) ([p2] body2))
		paramsMap = make(map[string]int)
		for _, arityForm := range node.Children[idx:] {
			if arityForm != nil && arityForm.Type == reader.NodeList && len(arityForm.Children) > 0 {
				pNode := arityForm.Children[0]
				if pNode != nil && pNode.Type == reader.NodeVector {
					pMap := r.extractParameters(pNode)
					for k, v := range pMap {
						paramsMap[k] = v
					}
					bodyNodes = append(bodyNodes, arityForm.Children[1:]...)
				}
			}
		}
	} else {
		return ""
	}

	if len(paramsMap) == 0 || len(bodyNodes) == 0 {
		return ""
	}

	for _, bodyNode := range bodyNodes {
		r.countParametersUses(bodyNode, paramsMap)
	}

	for mapKey, keyValue := range paramsMap {
		if keyValue <= 1 {
			delete(paramsMap, mapKey)
		}
	}

	if len(paramsMap) == 0 {
		return ""
	}

	parameters := slices.Collect(maps.Keys(paramsMap))
	return strings.Join(parameters, ", ")
}

func (r *MultipleEvaluationInMacrosRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *rules.Finding {
	if node != nil && node.Type == reader.NodeList && len(node.Children) > 1 &&
		node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol &&
		node.Children[0].Value == "defmacro" {
		smell := r.multipleEvaluation(node)
		if len(smell) != 0 {
			macroName := ""
			if node.Children[1] != nil {
				macroName = node.Children[1].Value
			}
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("The macro %s presents multiple calls to the input arguments %v without defining temporary local variables.", macroName, smell),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}
	return nil
}

func init() {
	defaultRule := &MultipleEvaluationInMacrosRule{
		Rule: rules.Rule{
			ID:          "multiple-evaluation-in-macros",
			Name:        "Multiple Evaluation in Macros",
			Description: "Inserting macro input arguments more than once without first binding them to a local, temporary variable violates macro best practices and leads to hidden side effects.",
			Severity:    rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}