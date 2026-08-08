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

func isHeavyResourceFunction(name string) bool {
	// True Semantic Analysis: checking for widely used Clojure ecosystem functions
	// that consume heavy resources like databases, HTTP clients, Redis, etc.
	knownFunctions := map[string]bool{
		// next.jdbc / clojure.java.jdbc
		"jdbc/execute!": true, "jdbc/query": true, "jdbc/insert!": true,
		"sql/execute!": true, "sql/query": true, "sql/insert!": true,
		// clj-http
		"client/get": true, "client/post": true, "client/put": true, "client/request": true,
		"http/get": true, "http/post": true, "http/put": true, "http/request": true,
		// taoensso.carmine (Redis)
		"car/wcar": true, "redis/wcar": true,
		// cognitect.aws (AWS/S3)
		"aws/invoke": true,
		// Kafka / Messaging
		"kafka/send": true, "kafka/send!": true,
		"producer/send": true, "producer/send!": true,
	}

	return knownFunctions[name]
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
	funcName := firstElement.Value

	// Ignore definitional and binding macros
	if funcName == "def" || funcName == "binding" || funcName == "let" || funcName == "fn" {
		return nil
	}

	if isHeavyResourceFunction(funcName) {
		for i := 1; i < len(node.Children); i++ {
			arg := node.Children[i]
			if arg.Type == reader.NodeSymbol {
				name := arg.Value
				if strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") && len(name) > 2 {
					if !isAllowedDynamicVar(name) {
						return &rules.Finding{
							RuleID:   r.Meta().ID,
							Message:  fmt.Sprintf("Passing dynamic variable `%s` to heavy resource function `%s`. Connection pools and stateful clients should be managed via dependency injection (like component or mount), not dynamic scope.", name, funcName),
							Filepath: filepath,
							Location: arg.Location,
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
