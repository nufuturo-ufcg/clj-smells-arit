package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type ImproperEmptinessCheckRule struct {
	Rule
}

func (r *ImproperEmptinessCheckRule) Meta() Rule {
	return Rule{
		ID:          "improper-emptiness-check",
		Name:        "Improper Emptiness Check",
		Description: "Detects improper ways of checking for collection emptiness. Recommends using `(seq coll)` for non-emptiness and `(empty? coll)` for emptiness.",
		Severity:    SeverityHint,
	}
}

func (r *ImproperEmptinessCheckRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	if len(node.Children) == 2 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "not" {
		isEmptyCall := node.Children[1]
		if isEmptyCall.Type == reader.NodeList && len(isEmptyCall.Children) == 2 &&
			isEmptyCall.Children[0].Type == reader.NodeSymbol && isEmptyCall.Children[0].Value == "empty?" {

			collectionExpr := isEmptyCall.Children[1].Value
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Improper emptiness check: `(not (empty? %s))`. Consider using `(seq %s)` for checking non-emptiness.", collectionExpr, collectionExpr),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	if len(node.Children) == 3 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "=" {
		arg1 := node.Children[1]
		arg2 := node.Children[2]

		var countNode *reader.RichNode
		var zeroNode *reader.RichNode

		if arg1.Type == reader.NodeList && len(arg1.Children) == 2 && arg1.Children[0].Type == reader.NodeSymbol && arg1.Children[0].Value == "count" && arg2.Type == reader.NodeNumber && arg2.Value == "0" {
			countNode = arg1
			zeroNode = arg2
		} else if arg2.Type == reader.NodeList && len(arg2.Children) == 2 && arg2.Children[0].Type == reader.NodeSymbol && arg2.Children[0].Value == "count" && arg1.Type == reader.NodeNumber && arg1.Value == "0" {
			countNode = arg2
			zeroNode = arg1
		}

		if countNode != nil && zeroNode != nil {

			collectionNode := countNode.Children[1]
			collectionExpr := collectionNode.Value
			zeroExpr := zeroNode.Value
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Improper emptiness check: `(= %s (count %s))`. Consider using `(empty? %s)`.", zeroExpr, collectionExpr, collectionExpr),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	if len(node.Children) == 3 && node.Children[0].Type == reader.NodeSymbol && (node.Children[0].Value == "<" || node.Children[0].Value == ">") {
		op := node.Children[0].Value
		arg1 := node.Children[1]
		arg2 := node.Children[2]

		var countNode *reader.RichNode
		var zeroNode *reader.RichNode
		var isLessThan bool

		if op == "<" && arg1.Type == reader.NodeNumber && arg1.Value == "0" && arg2.Type == reader.NodeList && len(arg2.Children) == 2 && arg2.Children[0].Type == reader.NodeSymbol && arg2.Children[0].Value == "count" {

			zeroNode = arg1
			countNode = arg2
			isLessThan = true
		} else if op == ">" && arg1.Type == reader.NodeList && len(arg1.Children) == 2 && arg1.Children[0].Type == reader.NodeSymbol && arg1.Children[0].Value == "count" && arg2.Type == reader.NodeNumber && arg2.Value == "0" {

			countNode = arg1
			zeroNode = arg2
			isLessThan = false
		}

		if countNode != nil && zeroNode != nil {

			collectionNode := countNode.Children[1]
			collectionExpr := collectionNode.Value
			zeroExpr := zeroNode.Value
			originalExpression := ""
			if isLessThan {
				originalExpression = fmt.Sprintf("(< %s (count %s))", zeroExpr, collectionExpr)
			} else {
				originalExpression = fmt.Sprintf("(> (count %s) %s)", collectionExpr, zeroExpr)
			}
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Improper emptiness check: `%s`. Consider using `(seq %s)` for checking non-emptiness.", originalExpression, collectionExpr),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

func init() {
	RegisterRule(&ImproperEmptinessCheckRule{
		Rule: Rule{
			ID:          "improper-emptiness-check",
			Name:        "Improper Emptiness Check",
			Description: "Detects improper ways of checking for collection emptiness. Recommends using `(seq coll)` for non-emptiness and `(empty? coll)` for emptiness.",
			Severity:    SeverityHint,
		},
	})
}
