package clojurespecific

import (
	"fmt"
	"strings"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type MisusedThreadingRule struct {
	rules.Rule
}

func (r *MisusedThreadingRule) Meta() rules.Rule {
	return r.Rule
}

func (r *MisusedThreadingRule) getDomain(funcName string) string {
	// Keyword access
	if strings.HasPrefix(funcName, ":") {
		return "MAP"
	}

	// Methods
	if strings.HasPrefix(funcName, ".") {
		return "INTEROP"
	}

	switch funcName {
	case "str", "name", "namespace", "pr-str":
		return "STRING"
	case "get", "first", "last", "nth", "assoc", "dissoc", "update", "update-in", "assoc-in", "keys", "vals", "select-keys":
		return "MAP"
	case "seq", "map", "filter", "reduce", "into", "concat", "reverse", "sort", "take", "drop", "rest", "next", "mapv", "filterv", "remove", "vector", "conj":
		return "SEQ"
	case "count", "pos?", "neg?", "zero?", "inc", "dec", "+", "-", "=", "boolean", "not", "and", "or", "<", ">", "<=", ">=", "empty?", "not-empty":
		return "LOGIC"
	case "slurp", "spit", "read-string", "println", "print", "prn":
		return "IO"
	}

	if strings.Contains(funcName, "clojure.string") || strings.HasPrefix(funcName, "str/") || strings.HasPrefix(funcName, "string/") {
		return "STRING"
	}

	if strings.Contains(funcName, "java.io") || strings.HasPrefix(funcName, "io/") || strings.Contains(funcName, "http") || strings.Contains(funcName, "json") || strings.Contains(funcName, "edn") {
		return "IO"
	}

	return "UNKNOWN"
}

func (r *MisusedThreadingRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol {
		return nil
	}
	
	if firstElement.Value != "->" && firstElement.Value != "->>" && firstElement.Value != "some->" && firstElement.Value != "some->>" {
		return nil
	}

	domainsFound := make(map[string]bool)
	hasLambda := false

	for i := 1; i < len(node.Children); i++ {
		step := node.Children[i]
		
		if step.Type == reader.NodeFnLiteral {
			hasLambda = true
		}

		var funcName string
		
		if step.Type == reader.NodeSymbol || step.Type == reader.NodeKeyword {
			funcName = step.Value
		} else if step.Type == reader.NodeList && len(step.Children) > 0 {
			firstStep := step.Children[0]
			if firstStep.Type == reader.NodeSymbol || firstStep.Type == reader.NodeKeyword {
				funcName = firstStep.Value
			} else if firstStep.Type == reader.NodeFnLiteral {
				hasLambda = true
			}
		}

		if funcName != "" {
			domain := r.getDomain(funcName)
			if domain != "UNKNOWN" {
				domainsFound[domain] = true
			}
		}
	}

	if len(domainsFound) >= 5 || hasLambda {
		message := fmt.Sprintf("Misused threading macro `%s`.", firstElement.Value)
		if hasLambda {
			message += " Using anonymous functions `#()` inside threading macros forces position and hurts readability. Consider using `as->` or a `let` block."
		} else {
			message += fmt.Sprintf(" The pipeline heavily mixes heterogeneous types or domains (found: %v). Consider breaking it down with `let` bindings for readability.", len(domainsFound))
		}

		return &rules.Finding{
			RuleID:   r.Meta().ID,
			Message:  message,
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Meta().Severity,
		}
	}

	return nil
}

func init() {
	defaultRule := &MisusedThreadingRule{
		Rule: rules.Rule{
			ID:          "misused-threading",
			Name:        "Misused Threading",
			Description: "Detects threading macros (->, ->>) that chain together completely heterogeneous operations, hurting readability.",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}
