package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

var primitiveTypeHints = map[string]struct{}{
	"String":  {},
	"Integer": {},
	"Long":    {},
	"Double":  {},
	"Float":   {},
	"Boolean": {},
	"Keyword": {},
	"Symbol":  {},
	"Number":  {},
	"Int":     {},
	"Bool":    {},
}

func isPrimitiveLike(paramNode *reader.RichNode) bool {
	if paramNode.Type == reader.NodeSymbol {
		paramName := paramNode.Value

		if paramName == "_" || paramName == "&" || strings.HasPrefix(paramName, "&") {
			return false
		}

		if paramNode.TypeHint == "" {
			return true
		}

		if _, isPrimitive := primitiveTypeHints[paramNode.TypeHint]; isPrimitive {
			return true
		}
	}
	return false
}

type PrimitiveObsessionParamsRule struct {
	MinConsecutivePrimitives int
}

func (r *PrimitiveObsessionParamsRule) Meta() Rule {

	minParams := r.MinConsecutivePrimitives
	if minParams <= 0 {
		minParams = 3
	}

	return Rule{
		ID:          "primitive-obsession",
		Name:        "Primitive Obsession",
		Description: fmt.Sprintf("Detects functions using primitive types where dedicated types (like records or maps) might be better. This includes functions with %d or more consecutive primitive/untyped parameters, or specific common primitive pairs.", minParams),
		Severity:    SeverityHint,
	}
}

func (r *PrimitiveObsessionParamsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}
	fnSymbol := node.Children[0]
	if fnSymbol.Type != reader.NodeSymbol || (fnSymbol.Value != "defn" && fnSymbol.Value != "defn-") {
		return nil
	}

	funcNameNode := node.Children[1]
	if funcNameNode.Type != reader.NodeSymbol {
		return nil
	}
	funcName := funcNameNode.Value

	var paramsVector *reader.RichNode
	paramStartIndex := 2
	if len(node.Children) > paramStartIndex && node.Children[paramStartIndex].Type == reader.NodeString {
		paramStartIndex++
	}
	if len(node.Children) > paramStartIndex && node.Children[paramStartIndex].Type == reader.NodeMap {
		paramStartIndex++
	}

	if len(node.Children) > paramStartIndex {
		firstParamsNode := node.Children[paramStartIndex]
		if firstParamsNode.Type == reader.NodeVector {

			paramsVector = firstParamsNode
		} else if firstParamsNode.Type == reader.NodeList && len(firstParamsNode.Children) > 0 {

			firstArityForm := firstParamsNode.Children[0]
			if firstArityForm.Type == reader.NodeList && len(firstArityForm.Children) > 0 && firstArityForm.Children[0].Type == reader.NodeVector {
				paramsVector = firstArityForm.Children[0]
			}
		}
	}

	if paramsVector == nil {
		return nil
	}

	minConsecutive := r.MinConsecutivePrimitives
	if minConsecutive <= 0 {
		minConsecutive = 3
	}

	params := paramsVector.Children

	if len(params) == 2 {
		param1 := params[0]
		param2 := params[1]
		if isPrimitiveLike(param1) && isPrimitiveLike(param2) {

			meta := r.Meta()
			return &Finding{
				RuleID:   meta.ID,
				Message:  fmt.Sprintf("Function '%s' takes exactly two primitive-like parameters ('%s', '%s'). Consider using a dedicated record or map to represent this concept.", funcName, param1.Value, param2.Value),
				Filepath: filepath,
				Location: param1.Location,
				Severity: meta.Severity,
			}
		}
	}

	consecutiveCount := 0
	var firstPrimitiveParam *reader.RichNode

	for _, paramNode := range params {
		if isPrimitiveLike(paramNode) {
			if consecutiveCount == 0 {
				firstPrimitiveParam = paramNode
			}
			consecutiveCount++
		} else {

			if consecutiveCount >= minConsecutive {
				meta := r.Meta()
				return &Finding{
					RuleID:   meta.ID,
					Message:  fmt.Sprintf("Function '%s' has %d consecutive primitive or untyped parameters starting at '%s'. Consider grouping them into a map or record.", funcName, consecutiveCount, firstPrimitiveParam.Value),
					Filepath: filepath,
					Location: firstPrimitiveParam.Location,
					Severity: meta.Severity,
				}
			}

			consecutiveCount = 0
			firstPrimitiveParam = nil
		}
	}

	if consecutiveCount >= minConsecutive {
		meta := r.Meta()
		return &Finding{
			RuleID:   meta.ID,
			Message:  fmt.Sprintf("Function '%s' has %d consecutive primitive or untyped parameters starting at '%s'. Consider grouping them into a map or record.", funcName, consecutiveCount, firstPrimitiveParam.Value),
			Filepath: filepath,
			Location: firstPrimitiveParam.Location,
			Severity: meta.Severity,
		}
	}

	return nil
}

func init() {

	defaultRule := &PrimitiveObsessionParamsRule{
		MinConsecutivePrimitives: 3,
	}
	RegisterRule(defaultRule)
}
