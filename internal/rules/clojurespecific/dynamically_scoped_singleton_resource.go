package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type DynamicallyScopedSingletonResourceRule struct {
	rules.Rule
}

func (r *DynamicallyScopedSingletonResourceRule) Meta() rules.Rule {
	return r.Rule
}

func isHeavyResourceFunction(node *reader.RichNode) bool {
	resolved := rules.ResolvedCall(node)
	if resolved == nil || resolved.Kind == reader.ResolutionUnresolved || resolved.Kind == reader.ResolutionLocal {
		return false
	}

	knownFunctions := map[string]struct{}{
		"clojure.java.jdbc/execute!": {}, "clojure.java.jdbc/query": {}, "clojure.java.jdbc/insert!": {},
		"next.jdbc/execute!": {}, "next.jdbc.sql/query": {}, "next.jdbc.sql/insert!": {},
		"clj-http.client/get": {}, "clj-http.client/post": {}, "clj-http.client/put": {}, "clj-http.client/request": {},
		"taoensso.carmine/wcar":       {},
		"cognitect.aws.client/invoke": {},
		"kafka/send":                  {}, "kafka/send!": {}, "producer/send": {}, "producer/send!": {},
	}
	_, ok := knownFunctions[resolved.CanonicalName]
	return ok
}

func (r *DynamicallyScopedSingletonResourceRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol {
		return nil
	}

	// Data-Flow Semantic Analysis
	// Ignore definitional and binding macros
	if firstElement.Value == "def" || firstElement.Value == "binding" || firstElement.Value == "let" || firstElement.Value == "fn" {
		return nil
	}

	if isHeavyResourceFunction(node) {
		for i := 1; i < len(node.Children); i++ {
			arg := node.Children[i]
			if arg.Type == reader.NodeSymbol {
				name := arg.Value
				if strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") && len(name) > 2 {
					if !isAllowedDynamicVar(name) {
						return rules.SetContextualFindingWithEvidence(&rules.Finding{
							RuleID:   r.Meta().ID,
							Message:  fmt.Sprintf("Passing dynamic variable `%s` to heavy resource function `%s`. Connection pools and stateful clients should be managed via dependency injection (like component or mount), not dynamic scope.", name, firstElement.Value),
							Filepath: filepath,
							Location: arg.Location,
							Severity: r.Meta().Severity,
						}, "Using a dynamic variable may be the deliberate contract of a library, fixture, or execution context.", "dynamic-binding-contract", "resource-ownership")
					}
				}
			}
		}
	}

	return nil
}

func init() {
	defaultRule := &DynamicallyScopedSingletonResourceRule{
		Rule: rules.Rule{
			ID:          "dynamically-scoped-singleton-resource",
			Name:        "Dynamically Scoped Singleton Resource",
			Description: "Flags the usage of dynamic variables (*var*) to manage heavy singleton resources like DB connections, which should be explicitly passed or managed by DI.",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}
