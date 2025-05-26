package rules

import (
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
)

type OveruseOfHighOrderFunctionsRule struct {
	Rule
}

func NewOveruseOfHighOrderFunctionsRule(cfg *config.Config) *OveruseOfHighOrderFunctionsRule {
	r := &OveruseOfHighOrderFunctionsRule{
		Rule: Rule{
			ID:          "overuse-of-high-order-functions",
			Name:        "Overuse of High-Order Functions",
			Description: "Detects specific patterns of overusing high-order functions like (map #(apply-twice ...)).",
			Severity:    SeverityWarning,
		},
	}
	return r
}

func (r *OveruseOfHighOrderFunctionsRule) Meta() Rule {
	return r.Rule
}

func (r *OveruseOfHighOrderFunctionsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) == 0 {

		return nil
	}
	firstChild := node.Children[0]

	if firstChild.Type != reader.NodeSymbol || firstChild.Value != "map" {

		return nil
	}

	if len(node.Children) < 2 {

		return nil
	}

	fnArgRichNode := node.Children[1]

	isFnLit := isFnLiteralRichNode(fnArgRichNode)

	if isFnLit {
		lambdaArgSymbol := getLambdaArgSymbolFromFnLiteralRichNode(fnArgRichNode)

		var effectiveBodyNode *reader.RichNode
		if fnArgRichNode.Type == reader.NodeFnLiteral {
			effectiveBodyNode = fnArgRichNode

		} else if fnArgRichNode.Type == reader.NodeList && len(fnArgRichNode.Children) > 0 && fnArgRichNode.Children[0].Value == "fn" {
			effectiveBodyNode = getLambdaBodyFromFnLiteralRichNode(fnArgRichNode)

		} else {

			return nil
		}

		if effectiveBodyNode == nil {

			return nil
		}

		isApplyTwiceCall := isCallToApplyTwiceRichNode(effectiveBodyNode, lambdaArgSymbol)

		if isApplyTwiceCall {

			return &Finding{
				RuleID:   r.ID,
				Message:  "This 'map' call with a lambda that uses 'apply-twice' might be an overuse of high-order functions. Consider simplifying the transformation.",
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}

		return nil

	} else {

		return nil
	}
}

func isFnLiteralRichNode(node *reader.RichNode) bool {

	result := node.Type == reader.NodeFnLiteral ||
		(node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "fn")

	return result
}

func getLambdaBodyFromFnLiteralRichNode(fnNode *reader.RichNode) *reader.RichNode {

	if fnNode.Type == reader.NodeFnLiteral {

		if len(fnNode.Children) > 0 {

			if fnNode.Children[0].Type == reader.NodeVector {
				if len(fnNode.Children) > 1 {

					return fnNode.Children[1]
				}

			} else {

				return fnNode.Children[0]
			}
		}

	} else if fnNode.Type == reader.NodeList && len(fnNode.Children) > 0 && fnNode.Children[0].Value == "fn" {

		children := fnNode.Children
		idx := 1
		if len(children) > idx && children[idx].Type == reader.NodeSymbol {

			idx++
		}
		if len(children) > idx && children[idx].Type == reader.NodeVector {

			idx++
		}
		if len(children) > idx {

			return children[idx]
		}

	}

	return nil
}

func getLambdaArgSymbolFromFnLiteralRichNode(fnNode *reader.RichNode) string {

	if fnNode.Type == reader.NodeFnLiteral {

		if len(fnNode.Children) > 0 && fnNode.Children[0].Type == reader.NodeVector {
			paramsVecNode := fnNode.Children[0]

			if len(paramsVecNode.Children) == 1 && paramsVecNode.Children[0].Type == reader.NodeSymbol {
				arg := paramsVecNode.Children[0].Value

				return arg
			}
		}

		return ""
	}

	if fnNode.Type == reader.NodeList && len(fnNode.Children) > 0 && fnNode.Children[0].Type == reader.NodeSymbol && fnNode.Children[0].Value == "fn" {

		children := fnNode.Children
		idx := 1
		if len(children) > idx && children[idx].Type == reader.NodeSymbol {
			idx++
		}
		if len(children) > idx && children[idx].Type == reader.NodeVector {
			paramsNode := children[idx]

			if len(paramsNode.Children) == 1 && paramsNode.Children[0].Type == reader.NodeSymbol {
				arg := paramsNode.Children[0].Value

				return arg
			}
		}
	}

	return ""
}

func isCallToApplyTwiceRichNode(bodyNode *reader.RichNode, lambdaArgSymbol string) bool {

	if bodyNode == nil {

		return false
	}

	var callElements []*reader.RichNode

	if bodyNode.Type == reader.NodeList || bodyNode.Type == reader.NodeFnLiteral {
		callElements = bodyNode.Children

	} else {

		return false
	}

	if len(callElements) == 0 {

		return false
	}

	var applyTwiceNode *reader.RichNode
	if callElements[0].Type == reader.NodeSymbol && callElements[0].Value == "apply-twice" {

		if len(callElements) < 3 {

			return false
		}
		applyTwiceNode = bodyNode

	} else if callElements[0].Type == reader.NodeList && len(callElements[0].Children) > 0 &&
		callElements[0].Children[0].Type == reader.NodeSymbol && callElements[0].Children[0].Value == "apply-twice" {

		applyTwiceNode = callElements[0]
		if len(applyTwiceNode.Children) < 3 {

			return false
		}

	} else {

		return false
	}

	actualCallArgs := applyTwiceNode.Children

	if len(actualCallArgs) < 3 {

		return false
	}

	lambdaValueNode := actualCallArgs[2]

	if lambdaValueNode.Type == reader.NodeSymbol {
		if lambdaArgSymbol != "" && lambdaValueNode.Value == lambdaArgSymbol {

			return true
		}

		if lambdaArgSymbol == "" && (lambdaValueNode.Value == "%" || lambdaValueNode.Value == "%1") {

			return true
		}
	}

	return false
}

func init() {
	RegisterRule(NewOveruseOfHighOrderFunctionsRule(nil))
}
