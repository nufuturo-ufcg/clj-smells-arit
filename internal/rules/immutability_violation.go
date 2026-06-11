package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

func init() {
	isSideEffectCall := Any(
		FirstChildValueEquals("println"),
		FirstChildValueEquals("print"),
		FirstChildValueEquals("prn"),
		FirstChildValueEquals("printf"),
		FirstChildValueEquals("spit"),
		FirstChildValueEquals("def"),
		FirstChildValueEquals("defonce"),
		FirstChildValueEquals("intern"),
	)

	NewRule("immutability-violation").
		Name("Immutability Violation").
		Description("Detects direct state mutation and violations of functional purity. Follows Clojure Style Guide recommendations for proper use of refs, atoms, agents, and avoiding global state mutation in local scopes.").
		Severity(SeverityWarning).
		When(IsList()).
		When(HasMinChildren(1)).
		When(ChildIsSymbol(0)).
		When(Any(
			// 1. Chamada direta de função de mutação
			Any(
				ChildValueEquals(0, "set!"),
				ChildValueEquals(0, "alter-var-root"),
				ChildValueEquals(0, "agent-send"),
				ChildValueEquals(0, "agent-send-off"),
				ChildValueEquals(0, "intern"),
				ChildValueEquals(0, "aset"),
			),
			// 2. def ou defonce dentro de escopo local
			All(
				Any(ChildValueEquals(0, "def"), ChildValueEquals(0, "defonce")),
				IsLocalScope(),
			),
			// 3. ref-set fora de dosync
			All(
				ChildValueEquals(0, "ref-set"),
				Not(IsInside("dosync")),
			),
			// 4. Chamada de reset!
			ChildValueEquals(0, "reset!"),
			// 5. Chamada de send ou send-off passando uma função com efeitos colaterais
			All(
				Any(ChildValueEquals(0, "send"), ChildValueEquals(0, "send-off")),
				HasMinChildren(3),
				ChildMatches(2, Any(
					isSideEffectCall,
					HasDescendant(isSideEffectCall),
				)),
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
