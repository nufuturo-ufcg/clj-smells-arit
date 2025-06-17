package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type LinearCollectionScanRule struct {
}

func (r *LinearCollectionScanRule) Meta() Rule {
	return Rule{
		ID:          "inappropriate-collection: linear-collection-scan",
		Name:        "Inappropriate Collection: Linear Collection Scan",
		Description: "Detects linear scans (e.g., using 'some' or 'filter' with an equality check on a key) on collections where a map lookup might be more efficient and idiomatic.",
		Severity:    SeverityHint,
	}
}

func (r *LinearCollectionScanRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol || (funcNode.Value != "some" && funcNode.Value != "filter") {
		return nil
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
				return nil
			}
		} else {
			return nil
		}
	} else {
		return nil
	}

	if fnBodyNode == nil || fnParamSymbol == "" {
		return nil
	}

	if fnBodyNode.Type != reader.NodeFnLiteral && fnBodyNode.Type != reader.NodeList {
		return nil
	}
	if len(fnBodyNode.Children) != 3 {
		return nil
	}

	eqNode := fnBodyNode.Children[0]
	if eqNode.Type != reader.NodeSymbol || (eqNode.Value != "=" && eqNode.Value != "==") {
		return nil
	}

	lhs := fnBodyNode.Children[1]
	rhs := fnBodyNode.Children[2]

	var keyAccessNode *reader.RichNode
	var otherSideNode *reader.RichNode

	lhsIsKeyAccess := isKeyAccess(lhs, fnParamSymbol)
	rhsIsKeyAccess := isKeyAccess(rhs, fnParamSymbol)

	if lhsIsKeyAccess && !rhsIsKeyAccess {
		keyAccessNode = lhs
		otherSideNode = rhs
	} else if rhsIsKeyAccess && !lhsIsKeyAccess {
		keyAccessNode = rhs
		otherSideNode = lhs
	} else {
		return nil
	}

	if otherSideNode.Type == reader.NodeSymbol && otherSideNode.Value == fnParamSymbol {
		return nil
	}

	meta := r.Meta()
	keyAccessStr := getKeyAccessString(keyAccessNode)
	finding := &Finding{
		RuleID:   meta.ID,
		Message:  fmt.Sprintf("Linear scan using '%s' with key access '%s'. Consider using a map for the collection and direct lookup (e.g., using 'get' or 'contains?') for better performance.", funcNode.Value, keyAccessStr),
		Filepath: filepath,
		Location: node.Location,
		Severity: meta.Severity,
	}
	return finding
}

func isKeyAccess(node *reader.RichNode, targetSymbol string) bool {
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

func getKeyAccessString(keyAccessNode *reader.RichNode) string {
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

func init() {

	RegisterRule(&LinearCollectionScanRule{})
}
