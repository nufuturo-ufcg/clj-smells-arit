package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type MiddleManRule struct {
	Rule
}

func (r *MiddleManRule) Meta() Rule {
	return r.Rule
}

func (r *MiddleManRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type == reader.NodeList && len(node.Children) > 1 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "defn" {

		funcNameNode := node.Children[1]

		paramsNodeIndex := 2
		bodyStartIndex := 3
		if len(node.Children) > paramsNodeIndex && node.Children[paramsNodeIndex].Type == reader.NodeString {

			paramsNodeIndex = 3
			bodyStartIndex = 4
		}

		if len(node.Children) <= paramsNodeIndex || node.Children[paramsNodeIndex].Type != reader.NodeVector {
			return nil
		}
		paramsNode := node.Children[paramsNodeIndex]
		outerParams := paramsNode.Children

		var actualOuterParams []*reader.RichNode
		for _, param := range outerParams {
			if !reader.IsVariadicMarker(param) {
				actualOuterParams = append(actualOuterParams, param)
			}
		}

		if len(node.Children) <= bodyStartIndex {
			return nil
		}
		bodyNodes := node.Children[bodyStartIndex:]

		var significantBodyNode *reader.RichNode
		for _, bNode := range bodyNodes {
			if bNode.Type != reader.NodeComment && bNode.Type != reader.NodeNewline {
				if significantBodyNode != nil {

					significantBodyNode = nil
					break
				}
				significantBodyNode = bNode
			}
		}

		if significantBodyNode == nil {
			return nil
		}

		if significantBodyNode.Type != reader.NodeList || len(significantBodyNode.Children) == 0 {
			return nil
		}

		innerCall := significantBodyNode.Children
		innerFuncNameNode := innerCall[0]
		innerArgs := innerCall[1:]

		if innerFuncNameNode.Type != reader.NodeSymbol {
			return nil
		}

		numOuter := len(actualOuterParams)
		numInner := len(innerArgs)

		if numOuter > 0 && numInner >= numOuter {
			match := r.checkParameterMatch(actualOuterParams, innerArgs, numOuter, numInner)

			if match {

				funcName := "?"
				if funcNameNode.Type == reader.NodeSymbol {
					funcName = funcNameNode.Value
				}
				innerFuncName := innerFuncNameNode.Value
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Function %q appears to be a 'Middle Man' delegating directly to %q. Consider using %q directly.", funcName, innerFuncName, innerFuncName),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}
	return nil
}

func (r *MiddleManRule) checkParameterMatch(outerParams, innerArgs []*reader.RichNode, numOuter, numInner int) bool {

	for i := 0; i < numOuter; i++ {
		outerParam := outerParams[i]
		innerArgIndex := numInner - numOuter + i
		innerArg := innerArgs[innerArgIndex]

		if !r.symbolsMatch(outerParam, innerArg) {
			return false
		}
	}

	if numInner > numOuter {
		prefixLen := numInner - numOuter
		for k := 0; k < prefixLen; k++ {
			prefixArg := innerArgs[k]
			if prefixArg.Type == reader.NodeSymbol {

				for _, outerParam := range outerParams {
					if r.symbolsMatch(prefixArg, outerParam) {
						return false
					}
				}
			}
		}
	}

	return true
}

func (r *MiddleManRule) symbolsMatch(sym1, sym2 *reader.RichNode) bool {
	if sym1.Type != reader.NodeSymbol || sym2.Type != reader.NodeSymbol {
		return false
	}

	if sym1.ResolvedDefinition != nil && sym2.ResolvedDefinition != nil {
		return sym1.ResolvedDefinition == sym2.ResolvedDefinition
	}

	return sym1.Value == sym2.Value && sym1.Value != "" && sym2.Value != ""
}

func init() {
	RegisterRule(&MiddleManRule{
		Rule: Rule{
			ID:          "middle-man",
			Name:        "Middle Man Function",
			Description: "Identifies functions that primarily delegate calls to other functions (often HOFs like map/filter) without adding significant logic, increasing unnecessary indirection.",
			Severity:    SeverityHint,
		},
	})

}
