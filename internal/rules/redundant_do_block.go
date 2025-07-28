package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type RedundantDoBlockRule struct {
	Rule
}

func (r *RedundantDoBlockRule) Meta() Rule {
	return Rule{
		ID:          "redundant-do-block",
		Name:        "Redundant `do` Block",
		Description: "Checks for redundant `do` blocks within forms that already imply sequential execution of their body or clauses.",
		Severity:    SeverityInfo,
	}
}

func getFnBodyStartIndex(parentChildren []*reader.RichNode, parentSymbol string) int {
	isDefnLike := parentSymbol == "defn" || parentSymbol == "defn-" || parentSymbol == "defmacro" || parentSymbol == "defmethod"
	isFnLike := parentSymbol == "fn"

	currentIndex := 1

	if isDefnLike {
		if currentIndex < len(parentChildren) && parentChildren[currentIndex].Type == reader.NodeSymbol {
			currentIndex++
		}
	} else if isFnLike {

		if currentIndex < len(parentChildren) && parentChildren[currentIndex].Type == reader.NodeSymbol &&
			currentIndex+1 < len(parentChildren) &&
			(parentChildren[currentIndex+1].Type == reader.NodeVector || parentChildren[currentIndex+1].Type == reader.NodeList) {
			currentIndex++
		}
	}

	if isDefnLike {
		if currentIndex < len(parentChildren) && parentChildren[currentIndex].Type == reader.NodeString {
			currentIndex++
		}
		if currentIndex < len(parentChildren) && parentChildren[currentIndex].Type == reader.NodeMap {
			currentIndex++
		}
	}

	if currentIndex < len(parentChildren) &&
		(parentChildren[currentIndex].Type == reader.NodeVector || parentChildren[currentIndex].Type == reader.NodeList) {
		return currentIndex + 1
	}

	return currentIndex
}

func (r *RedundantDoBlockRule) containsRecur(node *reader.RichNode) bool {
	if node == nil {
		return false
	}

	if node.Type == reader.NodeSymbol && node.Value == "recur" {
		return true
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		firstChild := node.Children[0]
		if firstChild.Type == reader.NodeSymbol && firstChild.Value == "recur" {
			return true
		}
	}

	for _, child := range node.Children {
		if r.containsRecur(child) {
			return true
		}
	}

	return false
}

func (r *RedundantDoBlockRule) hasMultipleExpressions(doNode *reader.RichNode) bool {
	if doNode == nil || doNode.Type != reader.NodeList || len(doNode.Children) <= 1 {
		return false
	}

	return len(doNode.Children) > 2
}

func (r *RedundantDoBlockRule) isInValidRefactoredContext(doNode *reader.RichNode, parent *reader.RichNode, doNodeIndex int) bool {
	if parent == nil || parent.Type != reader.NodeList || len(parent.Children) == 0 {
		return false
	}

	parentFirstElement := parent.Children[0]
	if parentFirstElement.Type != reader.NodeSymbol {
		return false
	}

	parentSymbol := parentFirstElement.Value

	if parentSymbol == "cond" && doNodeIndex >= 2 && doNodeIndex%2 == 0 {

		if r.hasMultipleExpressions(doNode) {

			return false
		}
	}

	return false
}

func (r *RedundantDoBlockRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol || firstElement.Value != "do" {
		return nil
	}

	var parent *reader.RichNode
	if pVal, ok := context["parent"]; ok {
		if pNode, pOk := pVal.(*reader.RichNode); pOk {
			parent = pNode
		}
	}

	if parent == nil || parent.Type != reader.NodeList || len(parent.Children) == 0 {
		return nil
	}

	doNodeIndex := -1
	for i, child := range parent.Children {
		if child == node {
			doNodeIndex = i
			break
		}
	}
	if doNodeIndex == -1 {
		return nil
	}

	parentFirstElement := parent.Children[0]
	if parentFirstElement.Type != reader.NodeSymbol {

		if parentFirstElement.Type == reader.NodeVector {
			var grandParent *reader.RichNode
			if gpVal, gpOk := context["grandparent"]; gpOk {
				if gpNode, gpNodeOk := gpVal.(*reader.RichNode); gpNodeOk {
					grandParent = gpNode
				}
			}

			if grandParent != nil && grandParent.Type == reader.NodeList && len(grandParent.Children) > 0 &&
				grandParent.Children[0].Type == reader.NodeSymbol {
				gpSymbol := grandParent.Children[0].Value
				switch gpSymbol {
				case "fn", "defn", "defn-", "defmacro", "defmethod", "proxy", "reify", "deftype", "defrecord", "extend-protocol", "extend-type":
					if doNodeIndex > 0 {
						return r.createFinding(node, parent, "arity list", filepath)
					}
				}
			}
		}
		return nil
	}

	parentSymbol := parentFirstElement.Value

	isRedundant := false
	redundantInForm := parentSymbol

	switch parentSymbol {
	case "let", "loop", "letfn", "binding", "with-local-vars", "with-open", "with-out-str", "with-in-str", "locking", "future", "promise", "testing", "comment", "doto", "doseq", "dotimes":

		if doNodeIndex >= 2 {
			isRedundant = true
		}

	case "when", "when-not":

		if doNodeIndex >= 2 {
			isRedundant = true
		}

	case "when-let", "when-some":

		if doNodeIndex >= 2 {
			isRedundant = true
		}

	case "if", "if-not":

		if doNodeIndex == 2 {
			if r.hasMultipleExpressions(node) {
				isRedundant = true
			}
		} else if doNodeIndex == 3 && len(parent.Children) == 4 {
			if r.hasMultipleExpressions(node) {
				isRedundant = true
			}
		}

	case "if-let", "if-some":

		if doNodeIndex == 2 {
			if r.hasMultipleExpressions(node) {
				isRedundant = true
			}
		} else if doNodeIndex == 3 && len(parent.Children) == 4 {
			if r.hasMultipleExpressions(node) {
				isRedundant = true
			}
		}

	case "fn", "defn", "defn-":

		bodyStartIndex := getFnBodyStartIndex(parent.Children, parentSymbol)
		if doNodeIndex >= bodyStartIndex {

			totalBodyExpressions := len(parent.Children) - bodyStartIndex
			if totalBodyExpressions == 1 {
				isRedundant = true
			}
		}

	case "defmacro":

		bodyStartIndex := getFnBodyStartIndex(parent.Children, parentSymbol)
		if doNodeIndex >= bodyStartIndex {
			totalBodyExpressions := len(parent.Children) - bodyStartIndex
			if totalBodyExpressions == 1 {
				isRedundant = true
			}
		}

	case "try":

		isTryBody := true
		for i := 1; i < doNodeIndex; i++ {
			if parent.Children[i].Type == reader.NodeList && len(parent.Children[i].Children) > 0 {
				childSymbolNode := parent.Children[i].Children[0]
				if childSymbolNode.Type == reader.NodeSymbol && (childSymbolNode.Value == "catch" || childSymbolNode.Value == "finally") {
					isTryBody = false
					break
				}
			}
		}
		if isTryBody && doNodeIndex >= 1 {
			isRedundant = true
		}

	case "catch":

		if doNodeIndex >= 3 {
			isRedundant = true
		}

	case "finally":

		if doNodeIndex >= 1 {
			isRedundant = true
		}

	case "cond":

		if doNodeIndex >= 2 && doNodeIndex%2 == 0 {

			if !r.hasMultipleExpressions(node) {

				isRedundant = true
			}
		}

	case "condp":

		if doNodeIndex >= 3 && doNodeIndex%2 == 1 {

			isRedundant = true
		}

	case "case":

		if doNodeIndex >= 2 && doNodeIndex%2 == 0 {

			isRedundant = true
		}
	}

	if isRedundant && !r.isInValidRefactoredContext(node, parent, doNodeIndex) {
		return r.createFinding(node, parent, redundantInForm, filepath)
	}

	return nil
}

func (r *RedundantDoBlockRule) createFinding(doNode, parentNode *reader.RichNode, parentFormName string, filepath string) *Finding {
	numDoChildren := len(doNode.Children) - 1

	message := fmt.Sprintf("Redundant `do` block found. The surrounding `%s` form already provides an implicit `do` for its body expressions.", parentFormName)
	if numDoChildren == 0 {
		message = fmt.Sprintf("Redundant empty `do` block found within `%s`. Consider removing it completely.", parentFormName)
	} else if numDoChildren == 1 {
		message = fmt.Sprintf("Redundant `do` block with a single expression found within `%s`. The `do` wrapper is unnecessary here.", parentFormName)
	}

	return &Finding{
		RuleID:   r.Meta().ID,
		Message:  message,
		Filepath: filepath,
		Location: doNode.Location,
		Severity: r.Meta().Severity,
	}
}

func init() {
	rule := &RedundantDoBlockRule{}
	RegisterRule(rule)
}
