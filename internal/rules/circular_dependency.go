package rules

import (
	"sort"  
	"strings" 
	"sync"  

	"github.com/thlaurentino/arit/internal/reader" 
)

type CyclicDependencyRule struct {
	Rule 
}

var (
	cyclicDepMutex sync.Mutex

	cyclicDepCallGraph map[string]map[string]map[string]*reader.RichNode

	cyclicDepFuncs map[string]map[string]bool

	cyclicDepChecked map[string]bool
)

func NewCyclicDependencyRule() *CyclicDependencyRule {
	return &CyclicDependencyRule{
		Rule: Rule{
			ID:          "cyclic-dependency", 
			Name:        "Cyclic Dependency", 
			Description: "Detects when two functions call each other, creating a direct mutual recursion cycle (e.g., A calls B, and B calls A).",
			Severity:    SeverityWarning,
		},
	}
}

func (r *CyclicDependencyRule) Meta() Rule {
	return r.Rule
}

func (r *CyclicDependencyRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	cyclicDepMutex.Lock()
	defer cyclicDepMutex.Unlock()

	if cyclicDepCallGraph[filepath] == nil {
		cyclicDepCallGraph[filepath] = make(map[string]map[string]*reader.RichNode) 
		cyclicDepFuncs[filepath] = make(map[string]bool)                            
		cyclicDepChecked[filepath] = false     
	}

	callGraph := cyclicDepCallGraph[filepath] 
	visitedFuncs := cyclicDepFuncs[filepath]  


	if node.Type == reader.NodeList && len(node.Children) > 1 {
		if node.Children[0].Type == reader.NodeSymbol {
			head := node.Children[0]

			if head.Value == "defn" || head.Value == "defn-" { // é uma função

				if node.Children[1].Type == reader.NodeSymbol {
					funcName := node.Children[1].Value 

					visitedFuncs[funcName] = true

					if callGraph[funcName] == nil {
						callGraph[funcName] = make(map[string]*reader.RichNode)
					}

					collectCalls(node, funcName, callGraph) //collect the calls from a function
				}
			}
		}
	}

	if len(visitedFuncs) >= 2 && !cyclicDepChecked[filepath] {
		cyclicDepChecked[filepath] = true

		finding := findMutualCycle(callGraph, visitedFuncs, filepath)

		delete(cyclicDepCallGraph, filepath)
		delete(cyclicDepFuncs, filepath)
		delete(cyclicDepChecked, filepath)
		
		return finding
	}

	return nil
}

func collectCalls(funcDefNode *reader.RichNode, callerName string, graph map[string]map[string]*reader.RichNode) {
	var walk func(*reader.RichNode)

	walk = func(node *reader.RichNode) {
		if node.Type == reader.NodeList && len(node.Children) > 0 {

			if node.Children[0].Type == reader.NodeSymbol {
				calleeName := node.Children[0].Value

				if calleeName != callerName {
					graph[callerName][calleeName] = node
				}
			}
		}

		for _, child := range node.Children {
			walk(child) 
		}
	}

	walk(funcDefNode)
}

func findMutualCycle(graph map[string]map[string]*reader.RichNode, visitedFuncs map[string]bool, filepath string) *Finding {
	reported := make(map[string]bool)
	for caller, callees := range graph {
		if !visitedFuncs[caller] {
			continue
		}
		for callee, location := range callees {
			if !visitedFuncs[callee] {
				continue
			}
			if otherCallees, ok := graph[callee]; ok {
				if _, mutual := otherCallees[caller]; mutual {
					pair := []string{caller, callee}
					sort.Strings(pair)
					pairKey := strings.Join(pair, "-")
					if !reported[pairKey] {
						return &Finding{
							RuleID:   "cyclic-dependency",
							Message:  "Mutual recursion detected between '" + caller + "' and '" + callee + "'.",
							Filepath: filepath,
							Location: location.Location,
							Severity: SeverityWarning,
						}
					}
				}
			}
		}
	}
	return nil
}
func init() {
	cyclicDepCallGraph = make(map[string]map[string]map[string]*reader.RichNode)
	cyclicDepFuncs = make(map[string]map[string]bool)
	cyclicDepChecked = make(map[string]bool)

	RegisterRule(NewCyclicDependencyRule())

}
