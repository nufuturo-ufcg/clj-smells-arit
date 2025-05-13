package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type ExternalDataCouplingRule struct {
}

func NewExternalDataCouplingRule() *ExternalDataCouplingRule {
	return &ExternalDataCouplingRule{}
}

func (r *ExternalDataCouplingRule) Meta() Rule {
	return Rule{
		ID:          "external-data-coupling",
		Name:        "External Data Coupling",
		Description: "Detects direct usage of unsanitized data from I/O operations in potentially problematic functions, such as string manipulation on unvalidated structured data fields.",
		Severity:    SeverityWarning,
	}
}

var ioFunctions = map[string]bool{
	"slurp":                      true,
	"line-seq":                   true,
	"clojure.java.io/reader":     true,
	"clojure.data.json/read-str": true,
	"json/read-str":              true,
	"clojure.edn/read-string":    true,
	"edn/read-string":            true,
	"clojure.xml/parse":          true,
	"http/get":                   true,
	"clj-http.client/get":        true,
	"http/post":                  true,
	"clj-http.client/post":       true,
}

var sanitizingFunctions = map[string]bool{
	"clojure.string/trim":        true,
	"str/trim":                   true,
	"clojure.string/triml":       true,
	"str/triml":                  true,
	"clojure.string/trimr":       true,
	"str/trimr":                  true,
	"clojure.string/escape":      true,
	"str/escape":                 true,
	"clojure.core/int":           true,
	"int":                        true,
	"clojure.core/long":          true,
	"long":                       true,
	"clojure.core/double":        true,
	"double":                     true,
	"clojure.core/boolean":       true,
	"boolean":                    true,
	"clojure.core/keyword":       true,
	"keyword":                    true,
	"clojure.spec.alpha/valid?":  true,
	"s/valid?":                   true,
	"clojure.spec.alpha/conform": true,
	"s/conform":                  true,
}

var problematicStringFunctions = map[string]bool{
	"clojure.string/includes?":    true,
	"str/includes?":               true,
	"clojure.string/starts-with?": true,
	"str/starts-with?":            true,
	"clojure.string/ends-with?":   true,
	"str/ends-with?":              true,
}

func (r *ExternalDataCouplingRule) isIoFunction(name string) bool {
	return ioFunctions[name]
}

func (r *ExternalDataCouplingRule) isSanitizingFunction(name string) bool {
	return sanitizingFunctions[name]
}

func (r *ExternalDataCouplingRule) isProblematicStringFunction(name string) bool {
	return problematicStringFunctions[name]
}

func (r *ExternalDataCouplingRule) isMapAccess(accessNode *reader.RichNode) (isAccess bool, dataSourceNode *reader.RichNode) {
	if accessNode.Type != reader.NodeList || len(accessNode.Children) < 1 {
		return false, nil
	}
	opNode := accessNode.Children[0]
	if opNode.Type == reader.NodeKeyword && len(accessNode.Children) >= 2 {
		return true, accessNode.Children[1]
	}
	if opNode.Type == reader.NodeList && len(opNode.Children) > 0 && opNode.Children[0].Type == reader.NodeSymbol && opNode.Children[0].Value == "keyword" && len(accessNode.Children) >= 2 {
		return true, accessNode.Children[1]
	}
	if opNode.Type == reader.NodeSymbol && opNode.Value == "get" && len(accessNode.Children) >= 2 {
		return true, accessNode.Children[1]
	}
	if opNode.Type == reader.NodeSymbol && strings.HasPrefix(opNode.Value, ":") && len(accessNode.Children) >= 2 {
		return true, accessNode.Children[1]
	}
	return false, nil
}

func (r *ExternalDataCouplingRule) findUnsanitizedIoDataSource(
	evalNode *reader.RichNode,
	currentSearchNode *reader.RichNode,
	visited map[*reader.RichNode]bool,
	definedInScope map[string]*reader.RichNode,
) (*reader.RichNode, bool) {
	if evalNode == nil {
		return nil, false
	}

	if _, ok := visited[evalNode]; ok {
		return nil, false
	}
	visited[evalNode] = true

	switch evalNode.Type {
	case reader.NodeList:
		if len(evalNode.Children) == 0 {
			return nil, false
		}
		funcSymNode := evalNode.Children[0]
		if funcSymNode.Type == reader.NodeSymbol {
			funcName := funcSymNode.Value
			if r.isIoFunction(funcName) {
				return evalNode, true
			}
			if r.isSanitizingFunction(funcName) {
				return nil, false
			}

			if funcSymNode.ResolvedDefinition != nil &&
				funcSymNode.ResolvedDefinition.Type == reader.NodeList &&
				len(funcSymNode.ResolvedDefinition.Children) > 0 &&
				funcSymNode.ResolvedDefinition.Children[0].Type == reader.NodeSymbol &&
				funcSymNode.ResolvedDefinition.Children[0].Value == "defn" {

				defnActualNode := funcSymNode.ResolvedDefinition
				var lastBodyExpr *reader.RichNode

				paramVecIndex := -1
				for i, child := range defnActualNode.Children {
					if child.Type == reader.NodeVector {
						paramVecIndex = i
						break
					}
				}

				if paramVecIndex != -1 && paramVecIndex < len(defnActualNode.Children)-1 {
					lastBodyExpr = defnActualNode.Children[len(defnActualNode.Children)-1]
				}

				if lastBodyExpr != nil {

					return r.findUnsanitizedIoDataSource(lastBodyExpr, lastBodyExpr, make(map[*reader.RichNode]bool), make(map[string]*reader.RichNode))
				}
			}
		}

		isAccess, dataSourceNode := r.isMapAccess(evalNode)
		if isAccess {

			mapAccessVisited := make(map[*reader.RichNode]bool)
			for k, v := range visited {
				mapAccessVisited[k] = v
			}

			return r.findUnsanitizedIoDataSource(dataSourceNode, currentSearchNode, mapAccessVisited, definedInScope)
		}
		return nil, false

	case reader.NodeSymbol:
		symbolName := evalNode.Value

		if definitionNode, ok := definedInScope[symbolName]; ok {
			newVisited := make(map[*reader.RichNode]bool)
			for k, v := range visited {
				newVisited[k] = v
			}

			return r.findUnsanitizedIoDataSource(definitionNode, currentSearchNode, newVisited, definedInScope)
		}

		return nil, false

	default:
		return nil, false
	}
}

func (r *ExternalDataCouplingRule) Check(rootNode *reader.RichNode, _ map[string]interface{}, filepath string) *Finding {
	var finding *Finding

	var walkFn func(node *reader.RichNode, parentLetBindings map[string]*reader.RichNode)
	walkFn = func(node *reader.RichNode, currentLetBindings map[string]*reader.RichNode) {
		if finding != nil {
			return
		}

		localBindings := make(map[string]*reader.RichNode)
		for k, v := range currentLetBindings {
			localBindings[k] = v
		}

		if node.Type == reader.NodeList && len(node.Children) > 1 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "let" {
			bindingsVec := node.Children[1]
			if bindingsVec.Type == reader.NodeVector {
				for i := 0; i+1 < len(bindingsVec.Children); i += 2 {
					boundSymNode := bindingsVec.Children[i]
					valueNode := bindingsVec.Children[i+1]
					if boundSymNode.Type == reader.NodeSymbol {
						localBindings[boundSymNode.Value] = valueNode
					}
				}
			}
		}

		if node.Type == reader.NodeList && len(node.Children) > 0 {
			funcCandidateNode := node.Children[0]
			if funcCandidateNode.Type == reader.NodeSymbol {
				problematicFuncName := funcCandidateNode.Value
				if r.isProblematicStringFunction(problematicFuncName) {
					for i := 1; i < len(node.Children); i++ {
						argNode := node.Children[i]
						visited := make(map[*reader.RichNode]bool)

						ioCallNode, isUnsanitized := r.findUnsanitizedIoDataSource(argNode, node, visited, localBindings)

						if isUnsanitized && ioCallNode != nil {
							ioFuncName := "unknown I/O function"
							if len(ioCallNode.Children) > 0 && ioCallNode.Children[0].Type == reader.NodeSymbol {
								ioFuncName = ioCallNode.Children[0].Value
							}
							argNodeRepresentation := argNode.Value
							if argNode.Type == reader.NodeList || argNode.Type == reader.NodeVector {
								argNodeRepresentation = string(argNode.Type)
							}

							message := fmt.Sprintf("Data from I/O function '%s' (at line %d) is used by problematic function '%s' via argument '%s' without apparent sanitization.", ioFuncName, ioCallNode.Location.StartLine, problematicFuncName, argNodeRepresentation)
							meta := r.Meta()
							finding = &Finding{
								RuleID:   meta.ID,
								Message:  message,
								Location: node.Location,
								Severity: meta.Severity,
								Filepath: filepath,
							}
							return
						}
					}
				}
			}
		}

		for _, child := range node.Children {
			walkFn(child, localBindings)
			if finding != nil {
				return
			}
		}
	}

	if rootNode != nil {
		walkFn(rootNode, make(map[string]*reader.RichNode))
	}

	return finding
}

func init() {

	RegisterRule(NewExternalDataCouplingRule())
}
