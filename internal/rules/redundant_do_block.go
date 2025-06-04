package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type RedundantDoBlockRule struct{}

func (r *RedundantDoBlockRule) Meta() Rule {
	return Rule{
		ID:          "redundant-do-block",
		Description: "Verifica blocos `do` redundantes dentro de formas que já implicam execução sequencial de seus corpos ou cláusulas.",
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

	parentFirstElement := parent.Children[0]
	if parentFirstElement.Type != reader.NodeSymbol {

		if parentFirstElement.Type == reader.NodeVector {

			doNodeIndexInArity := -1
			for i, child := range parent.Children {
				if child == node {
					doNodeIndexInArity = i
					break
				}
			}
			if doNodeIndexInArity > 0 {
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

						return r.createFinding(node, parent, "lista de aridade", filepath)
					}
				}
			}
		}
		return nil
	}
	parentSymbol := parentFirstElement.Value

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

		if doNodeIndex == 2 || (doNodeIndex == 3 && len(parent.Children) > 3) {
			isRedundant = true
		}
	case "if-let", "if-some":

		if doNodeIndex == 2 || (doNodeIndex == 3 && len(parent.Children) > 3) {
			isRedundant = true
		}
	case "fn", "defn", "defn-", "defmacro", "defmethod", "proxy", "reify", "deftype", "defrecord", "extend-protocol", "extend-type":
		bodyStartIndex := getFnBodyStartIndex(parent.Children, parentSymbol)
		if doNodeIndex >= bodyStartIndex {
			isRedundant = true
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
	case "cond", "condp":

		if parentSymbol == "cond" || parentSymbol == "condp" {
			if doNodeIndex >= 2 && doNodeIndex%2 == 0 {
				isRedundant = true
			}
		}
	case "case":

		if doNodeIndex >= 2 && doNodeIndex%2 == 0 {
			isRedundant = true
		}
	}

	if isRedundant {
		return r.createFinding(node, parent, redundantInForm, filepath)
	}

	return nil
}

func (r *RedundantDoBlockRule) createFinding(doNode, parentNode *reader.RichNode, parentFormName string, filepath string) *Finding {

	numDoChildren := len(doNode.Children) - 1

	message := fmt.Sprintf("Bloco `do` redundante encontrado. A forma `%s` circundante já fornece um `do` implícito para suas expressões de corpo.", parentFormName)
	if numDoChildren == 0 {
		message = fmt.Sprintf("Bloco `do` redundante e vazio encontrado dentro de `%s`. Considere removê-lo completamente.", parentFormName)
	} else if numDoChildren == 1 {
		message = fmt.Sprintf("Bloco `do` redundante com uma única expressão encontrado dentro de `%s`. O wrapper `do` é desnecessário aqui.", parentFormName)
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
	RegisterRule(&RedundantDoBlockRule{})
}
