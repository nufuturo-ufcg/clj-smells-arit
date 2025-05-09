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

		if len(node.Children) <= bodyStartIndex {
			return nil
		}
		bodyNodes := node.Children[bodyStartIndex:]

		var significantBodyNode *reader.RichNode
		for _, bNode := range bodyNodes {
			if bNode.Type != reader.NodeComment && bNode.Type != reader.NodeNewline {
				if significantBodyNode != nil {
					significantBodyNode = nil // Reset if more than one significant node
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
		// innerFuncNameStr := innerFuncNameNode.Value // REMOVED Unused variable

		numOuter := len(outerParams)
		numInner := len(innerArgs)

		// Revised Logic: Check if the LAST numOuter arguments of inner call match outer params, and numOuter > 0
		if numOuter > 0 && numInner >= numOuter {
			match := true
			for i := 0; i < numOuter; i++ {
				outerParam := outerParams[i]
				// Compare with the i-th element FROM THE END of innerArgs
				innerArgIndex := numInner - numOuter + i
				innerArg := innerArgs[innerArgIndex]
				outerParamDef := outerParam.ResolvedDefinition
				innerArgDef := innerArg.ResolvedDefinition

				if !(outerParam.Type == reader.NodeSymbol &&
					innerArg.Type == reader.NodeSymbol &&
					outerParamDef != nil &&
					innerArgDef == outerParamDef) {
					match = false
					break
				}
			}

			// Additional Check: Ensure none of the prefix arguments are outer parameters
			if match && numInner > numOuter {
				prefixLen := numInner - numOuter
				for k := 0; k < prefixLen; k++ {
					prefixArg := innerArgs[k]
					if prefixArg.Type == reader.NodeSymbol && prefixArg.ResolvedDefinition != nil {
						for _, outerParam := range outerParams {
							if outerParam.ResolvedDefinition != nil && prefixArg.ResolvedDefinition == outerParam.ResolvedDefinition {
								match = false // Prefix arg is an outer param, not a simple delegation
								break         // Break inner loop (outerParams)
							}
						}
					}
					if !match {
						break // Break outer loop (prefixArgs)
					}
				}
			}

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

/* // REMOVED Helper function for logging
// Helper function to get values from nodes for logging
func nodeValues(nodes []*reader.RichNode) []string {
	vals := make([]string, len(nodes))
	for i, n := range nodes {
		vals[i] = n.Value
	}
	return vals
}
*/

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
