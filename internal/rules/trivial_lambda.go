package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type TrivialLambdaRule struct {
	Rule
}

func (r *TrivialLambdaRule) Meta() Rule {
	return r.Rule
}

func (r *TrivialLambdaRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	var lambdaArgs []*reader.RichNode
	var lambdaBody []*reader.RichNode
	isFnLiteral := false

	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "fn" {

		argsIndex := 1
		bodyIndex := 2
		if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
			argsIndex = 2
			bodyIndex = 3
		}
		if len(node.Children) > argsIndex && node.Children[argsIndex].Type == reader.NodeVector {
			lambdaArgs = node.Children[argsIndex].Children
			if len(node.Children) > bodyIndex {
				lambdaBody = node.Children[bodyIndex:]
			}
		}
	} else if node.Type == reader.NodeFnLiteral {
		isFnLiteral = true

		lambdaBody = node.Children

	}

	if lambdaBody != nil {

		var innerCallNodes []*reader.RichNode
		if !isFnLiteral {
			if len(lambdaBody) == 1 && lambdaBody[0].Type == reader.NodeList {
				innerCallNodes = lambdaBody[0].Children
			}
		} else {
			innerCallNodes = lambdaBody
		}

		if len(innerCallNodes) > 0 && innerCallNodes[0].Type == reader.NodeSymbol {
			calledFuncSymbol := innerCallNodes[0]
			innerArgs := innerCallNodes[1:]

			argsMatch := false
			if isFnLiteral {

				if len(innerArgs) == 0 {

					argsMatch = true
				} else if len(innerArgs) == 1 && innerArgs[0].Type == reader.NodeSymbol && innerArgs[0].Value == "%" {

					argsMatch = true
				} else {

					allArgsAreNumbered := true
					maxN := 0
					for i, arg := range innerArgs {
						if arg.Type == reader.NodeSymbol && len(arg.Value) > 1 && arg.Value[0] == '%' {
							num := 0
							_, err := fmt.Sscan(arg.Value[1:], &num)
							if err != nil || num != i+1 {
								allArgsAreNumbered = false
								break
							}
							if num > maxN {
								maxN = num
							}
						} else {
							allArgsAreNumbered = false
							break
						}
					}

					if allArgsAreNumbered && len(innerArgs) == maxN {
						argsMatch = true
					}
				}
			} else {

				if len(lambdaArgs) == len(innerArgs) {
					match := true
					for i := range lambdaArgs {
						if !(lambdaArgs[i].Type == reader.NodeSymbol && innerArgs[i].Type == reader.NodeSymbol && lambdaArgs[i].Value == innerArgs[i].Value) {
							match = false
							break
						}
					}
					argsMatch = match
				}
			}

			if argsMatch {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Trivial lambda/fn. Consider using function %q directly.", calledFuncSymbol.Value),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}
	return nil
}

func init() {

	defaultRule := &TrivialLambdaRule{
		Rule: Rule{
			ID:          "trivial-lambda",
			Name:        "Trivial Lambda",
			Description: "Lambda or fn that merely calls another function with the same arguments can be replaced by the function itself.",
			Severity:    SeverityInfo,
		},
	}

	RegisterRule(defaultRule)
}
