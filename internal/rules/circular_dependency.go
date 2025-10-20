package rules

import (

	"github.com/thlaurentino/arit/internal/reader"
)

type CircularDependencyRule struct {
	Rule
}

func NewCircularDependencyRule() *CircularDependencyRule {
	return &CircularDependencyRule{
		Rule: Rule{
				ID:          "circular-dependency",
				Name:        "Circular Dependency",
				Description: "Detects circular dependencies between functions or modules. Circular dependencies make the codebase harder to maintain, understand, and test, as they create tight coupling and implicit execution order dependencies.",
				Severity:    SeverityWarning,
		},
	}
}


func (r *CircularDependencyRule) Meta() Rule {
	return r.Rule
}


func (r *CircularDependencyRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node == nil {	
		return nil
	}
	funcName := r.getFunctionName(node)
	return findCircularDependency(node, funcName, filepath)
}

func findCircularDependency(node *reader.RichNode, funcName string, filepath string) *Finding {
	if node == nil{ 
		return nil
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		first := node.Children[0]
		if first.Type == reader.NodeSymbol && first.Value == funcName {
			return &Finding{
				RuleID: "circular-dependency",
				Message: "Function appears to call itself (possible circular dependency)",
				Filepath: filepath,
				Location: node.Location,
				Severity: SeverityWarning,

			}
		}
	}

	for _, child := range node.Children {
    	finding := findCircularDependency(child, funcName, filepath)
    	if finding != nil {
        	return finding
    	}
	}
	return nil
}

func (r *CircularDependencyRule) getFunctionName(node *reader.RichNode) string {
	if node.Type == reader.NodeList && len(node.Children) > 1 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || node.Children[0].Value == "defn-") {
		if node.Children[1].Type == reader.NodeSymbol {
			return node.Children[1].Value
		}
	}
	return ""
}


