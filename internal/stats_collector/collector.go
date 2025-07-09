package stats_collector

import (
	"github.com/thlaurentino/arit/internal/reader"
)

type FunctionStats struct {
	FunctionName                  string
	LinesOfCode                   int
	ParameterCount                int
	MaxNestingDepth               int
	MaxMessageChain               int
	MaxConsecutivePrimitiveParams int
}

func Collect(rootNode *reader.RichNode) []FunctionStats {
	var stats []FunctionStats
	var findFunctions func(node *reader.RichNode)

	findFunctions = func(node *reader.RichNode) {
		if node == nil {
			return
		}

		if fnNode, fnName := IsFunctionDefinition(node); fnNode != nil {
			stats = append(stats, AnalyzeFunctionNode(fnNode, fnName))
		}

		for _, child := range node.Children {
			findFunctions(child)
		}
	}

	findFunctions(rootNode)
	return stats
}

func IsFunctionDefinition(node *reader.RichNode) (*reader.RichNode, string) {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil, ""
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return nil, ""
	}

	switch firstChild.Value {
	case "defn", "defn-":
		if len(node.Children) > 2 && node.Children[1].Type == reader.NodeSymbol {
			return node, node.Children[1].Value
		}
	case "fn":

		return node, "anonymous-fn"
	}

	if firstChild.Value == "fp/defsc" || firstChild.Value == "defsc" {
		if len(node.Children) > 2 && node.Children[1].Type == reader.NodeSymbol {
			return node, node.Children[1].Value
		}
	}

	if node.Type == reader.NodeFnLiteral {
		return node, "anonymous-literal-fn"
	}

	return nil, ""
}

func AnalyzeFunctionNode(fnNode *reader.RichNode, fnName string) FunctionStats {
	return FunctionStats{
		FunctionName:                  fnName,
		LinesOfCode:                   calculateLinesOfCode(fnNode),
		ParameterCount:                CountParameters(fnNode),
		MaxNestingDepth:               calculateNestingForBody(fnNode),
		MaxMessageChain:               calculateMaxMessageChain(fnNode),
		MaxConsecutivePrimitiveParams: countMaxConsecutivePrimitives(fnNode),
	}
}

func calculateLinesOfCode(fnNode *reader.RichNode) int {
	if fnNode.Location == nil || len(fnNode.Children) == 0 {
		return 0
	}

	lastChild := fnNode.Children[len(fnNode.Children)-1]
	if lastChild.Location == nil {
		return fnNode.Location.EndLine - fnNode.Location.StartLine + 1
	}

	return lastChild.Location.EndLine - fnNode.Location.StartLine + 1
}

func CountParameters(fnNode *reader.RichNode) int {
	if fnNode == nil || fnNode.Type != reader.NodeList {
		return 0
	}

	var paramsNode *reader.RichNode

	switch fnNode.Children[0].Value {
	case "defn", "defn-", "defsc", "defui", "deftest":
		idx := 2
		if len(fnNode.Children) > idx && fnNode.Children[idx].Type == reader.NodeString {
			idx++
		}
		if len(fnNode.Children) > idx && fnNode.Children[idx].Type == reader.NodeMap {
			idx++
		}
		if len(fnNode.Children) > idx && fnNode.Children[idx].Type == reader.NodeVector {
			paramsNode = fnNode.Children[idx]
		}
	case "fn":
		idx := 1
		if len(fnNode.Children) > idx && fnNode.Children[idx].Type == reader.NodeSymbol {
			idx++
		}
		if len(fnNode.Children) > idx && fnNode.Children[idx].Type == reader.NodeVector {
			paramsNode = fnNode.Children[idx]
		}
	}

	if paramsNode != nil && paramsNode.Type == reader.NodeVector {
		return reader.CountFunctionParameters(paramsNode)
	}

	return 0
}

func calculateNestingForBody(node *reader.RichNode) int {
	if node == nil {
		return 0
	}

	maxOverallCallStackDepth := 0

	var findDepth func(*reader.RichNode) int
	findDepth = func(n *reader.RichNode) int {
		if n == nil {
			return 0
		}

		maxChildCallDepth := 0
		isCall := false

		if n.Type == reader.NodeList && len(n.Children) > 0 {
			firstChildType := n.Children[0].Type
			if firstChildType == reader.NodeSymbol || firstChildType == reader.NodeKeyword {
				isCall = true
			}
		} else if n.Type == reader.NodeFnLiteral {
			isCall = false
			for _, child := range n.Children {
				depth := findDepth(child)
				if depth > maxChildCallDepth {
					maxChildCallDepth = depth
				}
			}
			return maxChildCallDepth
		}

		if isCall {
			for i := 1; i < len(n.Children); i++ {
				argDepth := findDepth(n.Children[i])
				if argDepth > maxChildCallDepth {
					maxChildCallDepth = argDepth
				}
			}
			return 1 + maxChildCallDepth
		} else if n.Type == reader.NodeList || n.Type == reader.NodeVector || n.Type == reader.NodeMap {
			for _, child := range n.Children {
				childDepth := findDepth(child)
				if childDepth > maxChildCallDepth {
					maxChildCallDepth = childDepth
				}
			}
			return maxChildCallDepth
		}

		return 0
	}

	for _, child := range node.Children {
		depth := findDepth(child)
		if depth > maxOverallCallStackDepth {
			maxOverallCallStackDepth = depth
		}
	}

	return maxOverallCallStackDepth
}

func calculateMaxMessageChain(fnNode *reader.RichNode) int {
	maxChain := 0

	var findChains func(node *reader.RichNode)
	findChains = func(node *reader.RichNode) {
		if node == nil {
			return
		}

		if node.Type == reader.NodeList && len(node.Children) >= 3 &&
			node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "get-in" &&
			node.Children[2].Type == reader.NodeVector {
			pathVector := node.Children[2]
			if len(pathVector.Children) > maxChain {
				maxChain = len(pathVector.Children)
			}
		}

		if node.Type == reader.NodeList && len(node.Children) > 1 &&
			node.Children[0].Type == reader.NodeSymbol && (node.Children[0].Value == "->" || node.Children[0].Value == "some->") {

			chainLen := len(node.Children) - 2
			if chainLen > maxChain {
				maxChain = chainLen
			}
		}

		for _, child := range node.Children {
			findChains(child)
		}
	}

	findChains(fnNode)
	return maxChain
}

func countMaxConsecutivePrimitives(fnNode *reader.RichNode) int {
	paramsNode := getParamsNode(fnNode)
	if paramsNode == nil {
		return 0
	}

	maxConsecutive := 0
	currentConsecutive := 0

	for _, param := range paramsNode.Children {
		if isPrimitiveLike(param) {
			currentConsecutive++
		} else {
			if currentConsecutive > maxConsecutive {
				maxConsecutive = currentConsecutive
			}
			currentConsecutive = 0
		}
	}
	if currentConsecutive > maxConsecutive {
		maxConsecutive = currentConsecutive
	}

	return maxConsecutive
}

func getParamsNode(fnNode *reader.RichNode) *reader.RichNode {

	for i, child := range fnNode.Children {
		if i > 0 && (fnNode.Children[0].Value == "defn" || fnNode.Children[0].Value == "defn-") {
			if child.Type == reader.NodeSymbol || child.Type == reader.NodeString || child.Type == reader.NodeMap {
				continue
			}
		}
		if child.Type == reader.NodeVector {
			return child
		}

		if child.Type == reader.NodeList {
			for _, arity := range child.Children {
				if arity.Type == reader.NodeList && len(arity.Children) > 0 && arity.Children[0].Type == reader.NodeVector {
					return arity.Children[0]
				}
			}
		}
	}
	return nil
}

func isPrimitiveLike(paramNode *reader.RichNode) bool {
	if paramNode.Type == reader.NodeSymbol {
		val := paramNode.Value
		if val == "&" || val == "_" {
			return false
		}
		return true
	}
	return false
}
