package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

var DefaultLazyContextFunctions = map[string]bool{
	"map":        true,
	"filter":     true,
	"remove":     true,
	"for":        true,
	"lazy-seq":   true,
	"mapcat":     true,
	"lazy-cat":   true,
	"keep":       true,
	"distinct":   true,
	"interpose":  true,
	"iterate":    true,
	"repeat":     true,
	"repeatedly": true,
	"cycle":      true,
}

var EagerConsumerFunctions = map[string]bool{
	"into":      true,
	"run!":      true,
	"doseq":     true,
	"reduce":    true,
	"transduce": true,
	"mapv":      true,
	"dorun":     true,
	"doall":     true,
	"sequence":  true,
}

var DefaultSideEffectFunctions = map[string]bool{
	"println":        true,
	"print":          true,
	"printf":         true,
	"prn":            true,
	"pr":             true,
	"swap!":          true,
	"reset!":         true,
	"add-watch":      true,
	"remove-watch":   true,
	"send":           true,
	"send-off":       true,
	"alter-var-root": true,
	"spit":           true,
	"aset":           true,
}

type LazySideEffectsRule struct {
	Rule
	LazyContextFuncs map[string]bool `json:"lazy_context_funcs" yaml:"lazy_context_funcs"`
	SideEffectFuncs  map[string]bool `json:"side_effect_funcs" yaml:"side_effect_funcs"`
}

func (r *LazySideEffectsRule) Meta() Rule {
	return Rule{
		ID:          "lazy-side-effects",
		Name:        "Lazy Side Effects",
		Description: "Detects potential side effects (like printing or state mutation) inside lazy sequence operations (map, filter, etc.). Side effects might not execute when expected. This rule ignores cases where the lazy operation is consumed by an eager function (e.g., 'into', 'run!', 'doseq').",
		Severity:    SeverityWarning,
	}
}

func isConsumedByEagerFunction(node *reader.RichNode, eagerFunctions map[string]bool) bool {

	return findEagerConsumerInAST(node, eagerFunctions, 0, 10)
}

func findEagerConsumerInAST(node *reader.RichNode, eagerFunctions map[string]bool, depth int, maxDepth int) bool {
	if depth > maxDepth || node == nil {
		return false
	}

	current := node
	for current != nil {

		if current.Type == reader.NodeList && len(current.Children) > 0 {
			firstChild := current.Children[0]
			if firstChild.Type == reader.NodeSymbol {
				if _, isEager := eagerFunctions[firstChild.Value]; isEager {

					if nodeContainsLazyOperation(current, node) {
						return true
					}
				}
			}
		}

		for _, child := range current.Children {
			if findEagerConsumerInAST(child, eagerFunctions, depth+1, maxDepth) {
				return true
			}
		}
		break
	}

	return false
}

func nodeContainsLazyOperation(container *reader.RichNode, target *reader.RichNode) bool {
	if container == target {
		return true
	}

	for _, child := range container.Children {
		if nodeContainsLazyOperation(child, target) {
			return true
		}
	}

	return false
}

const maxRecursionDepth = 10

func containsSideEffect(node *reader.RichNode, visited map[*reader.RichNode]bool, sideEffects map[string]bool, currentDepth int, maxDepth int) bool {
	if currentDepth > maxDepth {
		return false
	}
	if node == nil || visited[node] {
		return false
	}
	visited[node] = true

	if node.Type == reader.NodeFnLiteral {

		for _, child := range node.Children {
			if containsSideEffect(child, visited, sideEffects, currentDepth+1, maxDepth) {
				return true
			}
		}
		return false
	}

	if node.Type == reader.NodeSymbol {
		if _, isDirectSideEffect := sideEffects[node.Value]; isDirectSideEffect {
			return true
		}
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		funcNode := node.Children[0]
		if funcNode.Type == reader.NodeSymbol {

			if isDirectSideEffect := sideEffects[funcNode.Value]; isDirectSideEffect {
				return true
			}

			if funcNode.ResolvedDefinition != nil {
				definitionNode := funcNode.ResolvedDefinition
				var bodyNodesToAnalyze []*reader.RichNode

				if definitionNode.Type == reader.NodeList && len(definitionNode.Children) > 0 {
					defSymbol := definitionNode.Children[0]
					if defSymbol.Type == reader.NodeSymbol && (defSymbol.Value == "defn" || defSymbol.Value == "defn-") {
						bodyNodesToAnalyze = extractDefnBodyNodes(definitionNode)
					}
				}

				for _, bodyNode := range bodyNodesToAnalyze {
					newVisited := make(map[*reader.RichNode]bool)
					if containsSideEffect(bodyNode, newVisited, sideEffects, currentDepth+1, maxDepth) {
						return true
					}
				}
			}
		}
	}

	for _, child := range node.Children {
		if containsSideEffect(child, visited, sideEffects, currentDepth+1, maxDepth) {
			return true
		}
	}

	return false
}

func (r *LazySideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	isInEagerCtx, _ := context["isInEagerContext"].(bool)

	if isInEagerCtx {
		return nil
	}

	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return nil
	}
	lazyFuncName := funcNode.Value

	if _, isLazyContext := r.LazyContextFuncs[lazyFuncName]; !isLazyContext {
		return nil
	}

	if isConsumedByEagerFunction(node, EagerConsumerFunctions) {
		return nil
	}

	var funcArgNode *reader.RichNode
	if len(node.Children) >= 2 {
		if lazyFuncName == "for" && len(node.Children) > 2 && node.Children[1].Type == reader.NodeVector {
			for i := 2; i < len(node.Children); i++ {
				bodyExpr := node.Children[i]
				visited := make(map[*reader.RichNode]bool)
				if containsSideEffect(bodyExpr, visited, r.SideEffectFuncs, 0, maxRecursionDepth) {
					return r.createFinding(node, lazyFuncName, "body expression", filepath, isInEagerCtx)
				}
			}
			return nil
		} else if lazyFuncName == "lazy-seq" {
			for i := 1; i < len(node.Children); i++ {
				bodyExpr := node.Children[i]
				visited := make(map[*reader.RichNode]bool)
				if containsSideEffect(bodyExpr, visited, r.SideEffectFuncs, 0, maxRecursionDepth) {
					return r.createFinding(node, lazyFuncName, "body expression", filepath, isInEagerCtx)
				}
			}
			return nil
		} else {
			funcArgNode = node.Children[1]
		}
	}

	if funcArgNode == nil {
		return nil
	}

	var bodyToAnalyze *reader.RichNode
	funcNameStr := ""

	if funcArgNode.Type == reader.NodeFnLiteral {
		bodyToAnalyze = funcArgNode
		funcNameStr = "function literal (#(...))"
	} else if funcArgNode.Type == reader.NodeList && len(funcArgNode.Children) > 0 && funcArgNode.Children[0].Type == reader.NodeSymbol && funcArgNode.Children[0].Value == "fn" {
		bodyToAnalyze = funcArgNode
		funcNameStr = "function literal (fn ...)"
	} else if funcArgNode.Type == reader.NodeSymbol {
		funcNameStr = fmt.Sprintf("symbol '%s'", funcArgNode.Value)
		if funcArgNode.ResolvedDefinition != nil {
			bodyToAnalyze = funcArgNode.ResolvedDefinition
		} else {
			if _, isDirectSideEffect := r.SideEffectFuncs[funcArgNode.Value]; isDirectSideEffect {
				return r.createFinding(node, lazyFuncName, funcNameStr, filepath, isInEagerCtx)
			}
			return nil
		}
	} else {
		return nil
	}

	if bodyToAnalyze != nil {
		visited := make(map[*reader.RichNode]bool)
		if containsSideEffect(bodyToAnalyze, visited, r.SideEffectFuncs, 0, maxRecursionDepth) {
			return r.createFinding(node, lazyFuncName, funcNameStr, filepath, isInEagerCtx)
		}
	}

	return nil
}

func (r *LazySideEffectsRule) createFinding(node *reader.RichNode, lazyFuncName, funcSource string, filepath string, isInEagerCtx bool) *Finding {
	meta := r.Meta()

	messageSuffix := " Execution might be delayed or unexpected."
	if !isInEagerCtx {
		messageSuffix += " (Note: If this lazy call is ultimately consumed by an eager function like 'into' or 'run!', this warning might be a false positive)."
	}
	return &Finding{
		RuleID:   meta.ID,
		Message:  fmt.Sprintf("Function/symbol %s passed to lazy function '%s' may contain side effects.%s", funcSource, lazyFuncName, messageSuffix),
		Filepath: filepath,
		Location: node.Location,
		Severity: meta.Severity,
	}
}

func extractDefnBodyNodes(defnNode *reader.RichNode) []*reader.RichNode {
	bodyNodes := []*reader.RichNode{}
	if defnNode == nil || defnNode.Type != reader.NodeList || len(defnNode.Children) < 3 {
		return bodyNodes
	}

	currentIdx := 2

	if len(defnNode.Children) > currentIdx && defnNode.Children[currentIdx].Type == reader.NodeString {
		currentIdx++
	}

	if len(defnNode.Children) > currentIdx && defnNode.Children[currentIdx].Type == reader.NodeMap {
		currentIdx++
	}

	if len(defnNode.Children) <= currentIdx {
		return bodyNodes
	}

	if defnNode.Children[currentIdx].Type == reader.NodeList {
		multiArityList := defnNode.Children[currentIdx]
		for _, arityForm := range multiArityList.Children {

			if arityForm.Type == reader.NodeList && len(arityForm.Children) >= 2 && arityForm.Children[0].Type == reader.NodeVector {

				bodyNodes = append(bodyNodes, arityForm.Children[1:]...)
			}
		}
	} else if defnNode.Children[currentIdx].Type == reader.NodeVector {

		if len(defnNode.Children) > currentIdx+1 {
			bodyNodes = append(bodyNodes, defnNode.Children[currentIdx+1:]...)
		}
	}

	return bodyNodes
}

func init() {
	defaultRule := &LazySideEffectsRule{
		LazyContextFuncs: DefaultLazyContextFunctions,
		SideEffectFuncs:  DefaultSideEffectFunctions,
	}
	RegisterRule(defaultRule)
}
