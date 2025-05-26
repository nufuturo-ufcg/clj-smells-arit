package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type UnnecessaryAbstractionRule struct{}

func (r *UnnecessaryAbstractionRule) Meta() Rule {
	return Rule{
		ID:          "unnecessary-abstraction",
		Name:        "Unnecessary Abstraction (Single-Use Let Function)",
		Description: "Detects functions (`fn`) defined within a `let` that are called only once within the body of that same `let`. These functions can often be inlined or the logic simplified.",
		Severity:    SeverityHint,
	}
}

func (r *UnnecessaryAbstractionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	finding := r.checkSingleUseLetFn(node, context, filepath)
	if finding != nil {
		return finding
	}

	finding = r.checkSimpleDefn(node, context, filepath)
	if finding != nil {
		return finding
	}

	return nil
}

func (r *UnnecessaryAbstractionRule) checkSingleUseLetFn(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) < 3 || node.Children[0].Type != reader.NodeSymbol || node.Children[0].Value != "let" {
		return nil
	}

	bindingVectorNode := node.Children[1]
	if bindingVectorNode.Type != reader.NodeVector {
		return nil
	}

	letFnSymbols := make(map[string]*reader.RichNode)
	fnUsageCount := make(map[string]int)
	bindings := bindingVectorNode.Children
	if len(bindings)%2 != 0 {
		return nil
	}

	for i := 0; i < len(bindings); i += 2 {
		symNode := bindings[i]
		valNode := bindings[i+1]
		if symNode.Type == reader.NodeSymbol && valNode.Type == reader.NodeFnLiteral {
			letFnSymbols[symNode.Value] = symNode
			fnUsageCount[symNode.Value] = 0
		}
	}

	if len(letFnSymbols) == 0 {
		return nil
	}

	var findSymbolUsage func(targetNode *reader.RichNode, isInsideBinding bool)
	findSymbolUsage = func(targetNode *reader.RichNode, isInsideBinding bool) {
		if targetNode == nil {
			return
		}

		if isInsideBinding {

			for _, child := range targetNode.Children {
				isLetFnSymbol := false
				if child.Type == reader.NodeSymbol {
					if _, exists := letFnSymbols[child.Value]; exists {
						isLetFnSymbol = true
					}
				}
				if !isLetFnSymbol {
					findSymbolUsage(child, true)
				}
			}
			return
		}

		if targetNode.Type == reader.NodeSymbol {
			if _, exists := letFnSymbols[targetNode.Value]; exists {
				fnUsageCount[targetNode.Value]++
			}
		}

		for _, child := range targetNode.Children {
			findSymbolUsage(child, false)
		}
	}

	findSymbolUsage(bindingVectorNode, true)

	for _, bodyNode := range node.Children[2:] {
		findSymbolUsage(bodyNode, false)
	}

	meta := r.Meta()
	for fnSym, count := range fnUsageCount {
		if count == 1 {
			bindingNode := letFnSymbols[fnSym]
			message := fmt.Sprintf("The function '%s' defined in the 'let' is only used once. Consider inlining its logic.", fnSym)
			return &Finding{
				RuleID:   meta.ID,
				Message:  message,
				Filepath: filepath,

				Location: bindingNode.Location,
				Severity: meta.Severity,
			}
		}
	}

	return nil
}

func (r *UnnecessaryAbstractionRule) checkSimpleDefn(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) < 4 ||
		node.Children[0].Type != reader.NodeSymbol || node.Children[0].Value != "defn" ||
		node.Children[2].Type != reader.NodeVector {
		return nil
	}

	defnNameNode := node.Children[1]
	if defnNameNode.Type != reader.NodeSymbol {
		return nil
	}
	defnName := defnNameNode.Value

	paramsNode := node.Children[2]
	params := paramsNode.Children

	if len(params) == 0 {
		return nil
	}

	if len(node.Children[3:]) != 1 {
		return nil
	}
	bodyExpr := node.Children[3]

	meta := r.Meta()
	message := ""

	if bodyExpr.Type == reader.NodeList && len(bodyExpr.Children) == 2 &&
		bodyExpr.Children[0].Type == reader.NodeKeyword {
		keyNode := bodyExpr.Children[0]
		argNode := bodyExpr.Children[1]
		if argNode.Type == reader.NodeSymbol && len(params) > 0 && argNode.Value == params[0].Value {
			message = fmt.Sprintf("Function '%s' appears to only access key '%s' from its first argument. Consider inlining.", defnName, keyNode.Value)
		}
	} else if bodyExpr.Type == reader.NodeList && len(bodyExpr.Children) == 3 &&
		bodyExpr.Children[0].Type == reader.NodeSymbol && bodyExpr.Children[0].Value == "get" &&
		bodyExpr.Children[1].Type == reader.NodeSymbol && len(params) > 0 && bodyExpr.Children[1].Value == params[0].Value &&
		bodyExpr.Children[2].Type == reader.NodeKeyword {
		keyNode := bodyExpr.Children[2]
		message = fmt.Sprintf("Function '%s' appears to only get key '%s' from its first argument. Consider inlining.", defnName, keyNode.Value)
	}

	if message == "" && bodyExpr.Type == reader.NodeList && len(bodyExpr.Children) == len(params)+1 {
		delegatedCall := true

		for i, param := range params {
			callArg := bodyExpr.Children[i+1]
			if !(callArg.Type == reader.NodeSymbol && callArg.Value == param.Value) {
				delegatedCall = false
				break
			}
		}
		if delegatedCall {
			delegatedFnName := bodyExpr.Children[0].Value
			message = fmt.Sprintf("Function '%s' appears to only delegate the call to '%s' with the same arguments. Consider inlining or using '%s' directly.", defnName, delegatedFnName, delegatedFnName)
		}
	}

	if message != "" {
		return &Finding{
			RuleID:   meta.ID,
			Message:  message,
			Filepath: filepath,

			Location: defnNameNode.Location,
			Severity: meta.Severity,
		}
	}

	return nil
}

func init() {
	RegisterRule(&UnnecessaryAbstractionRule{})
}
