package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type MessageChainsRule struct {
	Rule
	MaxLength int `json:"max_length" yaml:"max_length"`
}

func (r *MessageChainsRule) Meta() Rule {
	return r.Rule
}

func checkNestedAccess(node *reader.RichNode) (bool, int) {
	if node.Type != reader.NodeList || len(node.Children) != 2 {
		return false, 0
	}
	first := node.Children[0]
	second := node.Children[1]

	isKeywordAccess := first.Type == reader.NodeKeyword
	isGetAccess := first.Type == reader.NodeSymbol && first.Value == "get" && len(node.Children) >= 3 && (node.Children[2].Type == reader.NodeKeyword || node.Children[2].Type == reader.NodeQuote || node.Children[2].Type == reader.NodeSymbol)

	if isKeywordAccess || isGetAccess {
		var nestedExpr *reader.RichNode
		if isKeywordAccess {
			nestedExpr = second
		} else {
			nestedExpr = node.Children[1]
		}
		isChain, length := checkNestedAccess(nestedExpr)
		if isChain {
			return true, length + 1
		}

		return true, 1
	}

	return false, 0
}

func checkGetInAccess(node *reader.RichNode) (bool, int) {
	if node.Type == reader.NodeList && len(node.Children) >= 3 &&
		node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "get-in" &&
		node.Children[2].Type == reader.NodeVector {
		pathVector := node.Children[2]

		return true, len(pathVector.Children)
	}
	return false, 0
}

func checkThreadFirst(node *reader.RichNode) (bool, int) {
	if node.Type == reader.NodeList && len(node.Children) > 1 &&
		node.Children[0].Type == reader.NodeSymbol && (node.Children[0].Value == "->" || node.Children[0].Value == "some->") {

		return true, len(node.Children) - 2
	}
	return false, 0
}

func checkGetChain(node *reader.RichNode, maxLength int) *Finding {
	if node.Type != reader.NodeList || len(node.Children) < 2 || node.Children[0].Type != reader.NodeSymbol || node.Children[0].Value != "get" {
		return nil
	}

	var find func(*reader.RichNode, int) (int, *Finding)
	find = func(n *reader.RichNode, depth int) (int, *Finding) {
		if n.Type != reader.NodeList || len(n.Children) < 2 || n.Children[0].Type != reader.NodeSymbol || n.Children[0].Value != "get" {
			return depth, nil
		}
		currentDepth, recFinding := find(n.Children[1], depth+1)
		if recFinding != nil {
			return currentDepth, recFinding
		}
		if currentDepth >= maxLength {

			return currentDepth, &Finding{
				Message:  fmt.Sprintf("Nested 'get' chain detected with depth %d (max %d).", currentDepth, maxLength),
				Location: node.Location,
			}
		}
		return currentDepth, nil
	}

	_, finding := find(node, 0)
	return finding
}

func checkGetInChain(node *reader.RichNode, maxLength int) *Finding {
	if node.Type == reader.NodeList && len(node.Children) == 3 &&
		node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "get-in" &&
		node.Children[2].Type == reader.NodeVector {

		keysNode := node.Children[2]
		chainLength := len(keysNode.Children)
		if chainLength >= maxLength {
			return &Finding{
				Message:  fmt.Sprintf("'get-in' chain detected with depth %d (max %d).", chainLength, maxLength),
				Location: node.Location,
			}
		}
	}
	return nil
}

func (r *MessageChainsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	getFinding := checkGetChain(node, r.MaxLength)
	if getFinding != nil {
		getFinding.RuleID = r.ID
		getFinding.Filepath = filepath
		getFinding.Severity = r.Severity
		return getFinding
	}

	getInFinding := checkGetInChain(node, r.MaxLength)
	if getInFinding != nil {
		getInFinding.RuleID = r.ID
		getInFinding.Filepath = filepath
		getInFinding.Severity = r.Severity
		return getInFinding
	}

	if node.Type == reader.NodeList && len(node.Children) > 1 && node.Children[0].Type == reader.NodeSymbol {
		macroName := node.Children[0].Value
		isThreadMacro := (macroName == "->" || macroName == "->>")

		if isThreadMacro {

			dataAccessChainLength := 0
			for i := 2; i < len(node.Children); i++ {
				stepNode := node.Children[i]
				isDataAccess := false
				if stepNode.Type == reader.NodeKeyword {
					isDataAccess = true
				} else if stepNode.Type == reader.NodeList && len(stepNode.Children) > 0 && stepNode.Children[0].Type == reader.NodeSymbol {
					funcName := stepNode.Children[0].Value
					if funcName == "get" || funcName == "get-in" || funcName == "." {
						isDataAccess = true
					}
				}
				if isDataAccess {
					dataAccessChainLength++
				}
			}

			if dataAccessChainLength >= r.MaxLength {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Potential data access message chain using '%s' with data access depth %d (max %d). Consider Law of Demeter.", macroName, dataAccessChainLength, r.MaxLength),
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
	defaultRule := &MessageChainsRule{
		Rule: Rule{
			ID:          "message-chains",
			Name:        "Message Chains",
			Description: "Avoid long chains of method calls or accesses (e.g., using '->', 'get-in', or nested keyword access). This increases coupling and violates the Law of Demeter.",
			Severity:    SeverityWarning,
		},
		MaxLength: 5,
	}

	RegisterRule(defaultRule)
}
