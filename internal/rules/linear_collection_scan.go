package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type LinearCollectionScanRule struct{}

func (r *LinearCollectionScanRule) Meta() Rule {
	return Rule{
		ID:          "linear-collection-scan",
		Name:        "Linear Collection Scan",
		Description: "Detects inefficient linear scanning patterns in collections, such as using 'some' or 'filter' with key access patterns that could be optimized with maps.",
		Severity:    SeverityHint,
	}
}

// Funções auxiliares para detecção de linear scan
func isKeyAccessLinearScan(node *reader.RichNode, targetSymbol string) bool {
	if node == nil {
		return false
	}

	if node.Type == reader.NodeList && len(node.Children) == 2 &&
		node.Children[0].Type == reader.NodeKeyword &&
		node.Children[1].Type == reader.NodeSymbol {
		if node.Children[1].Value == targetSymbol {
			return true
		}
	}

	if node.Type == reader.NodeList && len(node.Children) >= 3 &&
		node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "get" &&
		node.Children[1].Type == reader.NodeSymbol {
		if node.Children[1].Value == targetSymbol {
			return true
		}
	}
	return false
}

func getKeyAccessStringLinearScan(keyAccessNode *reader.RichNode) string {
	if keyAccessNode == nil {
		return "<unknown key access>"
	}

	if keyAccessNode.Type == reader.NodeList && len(keyAccessNode.Children) == 2 && keyAccessNode.Children[0].Type == reader.NodeKeyword {
		return keyAccessNode.Children[0].Value
	}

	if keyAccessNode.Type == reader.NodeList && len(keyAccessNode.Children) >= 3 && keyAccessNode.Children[0].Type == reader.NodeSymbol && keyAccessNode.Children[0].Value == "get" {
		keyNode := keyAccessNode.Children[2]
		if keyNode.Type == reader.NodeKeyword || keyNode.Type == reader.NodeString || keyNode.Type == reader.NodeSymbol {
			return keyNode.Value
		}
	}

	return "<complex key access>"
}

func (r *LinearCollectionScanRule) checkLinearScan(node *reader.RichNode, funcName string) (string, bool) {
	if len(node.Children) < 3 {
		return "", false
	}

	fnArgNode := node.Children[1]
	var fnBodyNode *reader.RichNode
	var fnParamSymbol string

	if fnArgNode.Type == reader.NodeFnLiteral && len(fnArgNode.Children) > 0 {
		fnBodyNode = fnArgNode
		fnParamSymbol = "%"
	} else if fnArgNode.Type == reader.NodeList && len(fnArgNode.Children) > 1 &&
		fnArgNode.Children[0].Type == reader.NodeSymbol && fnArgNode.Children[0].Value == "fn" {
		paramIndex := 1
		if len(fnArgNode.Children) > paramIndex && fnArgNode.Children[paramIndex].Type == reader.NodeSymbol {
			paramIndex++
		}
		if len(fnArgNode.Children) > paramIndex && fnArgNode.Children[paramIndex].Type == reader.NodeVector && len(fnArgNode.Children[paramIndex].Children) > 0 {
			paramsVec := fnArgNode.Children[paramIndex]
			if paramsVec.Children[0].Type == reader.NodeSymbol {
				fnParamSymbol = paramsVec.Children[0].Value
			}
			if len(fnArgNode.Children) > paramIndex+1 {
				fnBodyNode = fnArgNode.Children[len(fnArgNode.Children)-1]
			} else {
				return "", false
			}
		} else {
			return "", false
		}
	} else {
		return "", false
	}

	if fnBodyNode == nil || fnParamSymbol == "" {
		return "", false
	}

	if fnBodyNode.Type != reader.NodeFnLiteral && fnBodyNode.Type != reader.NodeList {
		return "", false
	}
	if len(fnBodyNode.Children) != 3 {
		return "", false
	}

	eqNode := fnBodyNode.Children[0]
	if eqNode.Type != reader.NodeSymbol || (eqNode.Value != "=" && eqNode.Value != "==") {
		return "", false
	}

	lhs := fnBodyNode.Children[1]
	rhs := fnBodyNode.Children[2]

	var keyAccessNode *reader.RichNode
	var otherSideNode *reader.RichNode

	lhsIsKeyAccess := isKeyAccessLinearScan(lhs, fnParamSymbol)
	rhsIsKeyAccess := isKeyAccessLinearScan(rhs, fnParamSymbol)

	if lhsIsKeyAccess && !rhsIsKeyAccess {
		keyAccessNode = lhs
		otherSideNode = rhs
	} else if rhsIsKeyAccess && !lhsIsKeyAccess {
		keyAccessNode = rhs
		otherSideNode = lhs
	} else {
		return "", false
	}

	if otherSideNode.Type == reader.NodeSymbol && otherSideNode.Value == fnParamSymbol {
		return "", false
	}

	keyAccessStr := getKeyAccessStringLinearScan(keyAccessNode)
	message := fmt.Sprintf("Linear scan using '%s' with key access '%s'. Consider using a map for the collection and direct lookup (e.g., using 'get' or 'contains?') for better performance.", funcName, keyAccessStr)

	return message, true
}

func (r *LinearCollectionScanRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return nil
	}

	funcName := funcNode.Value

	switch funcName {
	case "some":
		// Detectar linear scan com key access
		if linearMessage, hasLinearScan := r.checkLinearScan(node, "some"); hasLinearScan {
			meta := r.Meta()
			return &Finding{
				RuleID:   meta.ID,
				Message:  linearMessage,
				Filepath: filepath,
				Location: node.Location,
				Severity: meta.Severity,
			}
		}
	case "filter":
		// Detectar linear scan com key access
		if linearMessage, hasLinearScan := r.checkLinearScan(node, "filter"); hasLinearScan {
			meta := r.Meta()
			return &Finding{
				RuleID:   meta.ID,
				Message:  linearMessage,
				Filepath: filepath,
				Location: node.Location,
				Severity: meta.Severity,
			}
		}
	}

	return nil
}

func init() {
	RegisterRule(&LinearCollectionScanRule{})
}
