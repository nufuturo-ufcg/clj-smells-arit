package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type MisuseOfDynamicScopeRule struct {
	rules.Rule
}

func (r *MisuseOfDynamicScopeRule) Meta() rules.Rule {
	return r.Rule
}

func isAllowedDynamicVar(name string) bool {
	switch name {
	case "*out*", "*err*", "*in*", "*ns*", "*warn-on-reflection*", "*file*", "*compile-path*", "*command-line-args*", "*agent*", "*math-context*", "*print-length*", "*print-level*", "*data-readers*", "*default-data-reader-fn*":
		return true
	}
	return false
}

func isExplicitResourceVar(name string) bool {
	lower := strings.ToLower(name)
	lower = strings.TrimPrefix(lower, "*")
	lower = strings.TrimSuffix(lower, "*")
	
	parts := strings.Split(lower, "-")
	keywords := map[string]bool{
		"db": true, "conn": true, "pool": true, "tx": true, "transaction": true, 
		"client": true, "producer": true, "socket": true, "ds": true, 
		"redis": true, "kafka": true, "s3": true, "http": true,
		"writer": true, "session": true,
	}
	
	for _, part := range parts {
		if keywords[part] {
			return true
		}
	}
	return false
}

func (r *MisuseOfDynamicScopeRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol {
		return nil
	}

	if firstElement.Value == "def" && len(node.Children) > 1 {
		var name string
		var loc *reader.Location
		
		for i := 1; i < len(node.Children); i++ {
			if node.Children[i].Type == reader.NodeSymbol {
				name = node.Children[i].Value
				loc = node.Children[i].Location
				break
			}
		}

		if name != "" && strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") && len(name) > 2 {
			if !isAllowedDynamicVar(name) && !isExplicitResourceVar(name) {
				return &rules.Finding{
					RuleID:   r.Meta().ID,
					Message:  fmt.Sprintf("Defining custom dynamic variable `%s`. Dynamic scope is often misused for passing business context, which obfuscates data flow and causes bugs in async boundaries.", name),
					Filepath: filepath,
					Location: loc,
					Severity: r.Meta().Severity,
				}
			}
		}
	}

	// Flag binding of custom dynamic vars
	if firstElement.Value == "binding" && len(node.Children) > 1 {
		bindings := node.Children[1]
		if bindings.Type == reader.NodeVector {
			for i := 0; i < len(bindings.Children); i += 2 {
				varNode := bindings.Children[i]
				if varNode.Type == reader.NodeSymbol {
					name := varNode.Value
					if !isAllowedDynamicVar(name) && !isExplicitResourceVar(name) {
						return &rules.Finding{
							RuleID:   r.Meta().ID,
							Message:  fmt.Sprintf("Binding custom dynamic variable `%s`. This is a thread-local change and context will be lost if delegating to future/agent/go blocks.", name),
							Filepath: filepath,
							Location: varNode.Location,
							Severity: r.Meta().Severity,
						}
					}
				}
			}
		}
	}

	return nil
}

func init() {
	defaultRule := &MisuseOfDynamicScopeRule{
		Rule: rules.Rule{
			ID:          "misuse-of-dynamic-scope",
			Name:        "Misuse of Dynamic Scope",
			Description: "Flags the definition and usage of custom dynamic variables (*var*), which are often misused to pass business parameters implicitly, hiding dependencies and breaking thread boundaries.",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}
