package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

var mutatingSymbols = map[string]struct{}{
	"set!":           {},
	"alter-var-root": {},
	"agent-send":     {},
	"agent-send-off": {},
	"intern":         {},
	"aset":           {},
}

var sideEffectSymbols = map[string]struct{}{
	"println": {},
	"print":   {},
	"prn":     {},
	"printf":  {},
	"spit":    {},
	"def":     {},
	"defonce": {},
	"intern":  {},
}

func init() {
	NewRule("immutability-violation").
		Name("Immutability Violation").
		Description("Detects direct state mutation and violations of functional purity. Follows Clojure Style Guide recommendations for proper use of refs, atoms, agents, and avoiding global state mutation in local scopes.").
		Severity(SeverityWarning).
		When(IsList()).
		When(HasMinChildren(1)).
		When(ChildMatches(0, IsSymbol())).
		When(Any(
			// 1. Chamada direta de função de mutação
			ChildMatches(0, func(n *reader.RichNode, _ map[string]interface{}, _ string) bool {
				_, ok := mutatingSymbols[n.Value]
				return ok
			}),
			// 2. def ou defonce dentro de escopo local
			All(
				ChildMatches(0, Any(ValueEquals("def"), ValueEquals("defonce"))),
				IsLocalScope(),
			),
			// 3. ref-set fora de dosync
			All(
				ChildMatches(0, ValueEquals("ref-set")),
				Not(IsInside("dosync")),
			),
			// 4. Chamada de reset!
			ChildMatches(0, ValueEquals("reset!")),
			// 5. Chamada de send ou send-off passando uma função com efeitos colaterais
			All(
				ChildMatches(0, Any(ValueEquals("send"), ValueEquals("send-off"))),
				HasMinChildren(3),
				func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
					return containsSideEffects(node.Children[2])
				},
			),
		)).
		MessageFunc(func(node *reader.RichNode, _ map[string]interface{}) string {
			sym := node.Children[0].Value
			switch sym {
			case "def", "defonce":
				return fmt.Sprintf("Found `%s` inside a local scope. This mutates global state and should be avoided.", sym)
			case "ref-set":
				return "Found `ref-set` outside of `dosync`. Use `dosync` to ensure transactional safety with refs."
			case "reset!":
				return "Found `reset!`. Consider using `swap!` for atomic updates based on current value."
			case "send", "send-off":
				return "Found side effects in function passed to agent. Agent functions should be pure."
			default:
				return fmt.Sprintf("Found state mutation function call: `%s`. This can lead to side effects and violates immutability principles.", sym)
			}
		}).
		SeverityFunc(func(node *reader.RichNode, _ map[string]interface{}, defaultSev Severity) Severity {
			if len(node.Children) > 0 && node.Children[0].Value == "reset!" {
				return SeverityInfo
			}
			return defaultSev
		}).
		Register()
}

func containsSideEffects(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		symbol := node.Children[0]
		if symbol.Type == reader.NodeSymbol {
			if _, hasSideEffect := sideEffectSymbols[symbol.Value]; hasSideEffect {
				return true
			}
		}
	}
	for _, child := range node.Children {
		if containsSideEffects(child) {
			return true
		}
	}
	return false
}
