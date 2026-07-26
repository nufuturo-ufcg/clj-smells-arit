package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

func init() {
	isSideEffectCall := rules.Any(
		rules.FirstChildValueEquals("println"),
		rules.FirstChildValueEquals("print"),
		rules.FirstChildValueEquals("prn"),
		rules.FirstChildValueEquals("printf"),
		rules.FirstChildValueEquals("spit"),
		rules.FirstChildValueEquals("def"),
		rules.FirstChildValueEquals("defonce"),
		rules.FirstChildValueEquals("intern"),
	)

	rules.NewRule("immutability-violation").
		Name("Immutability Violation").
		Description("Detects direct state mutation and violations of functional purity. Follows Clojure Style Guide recommendations for proper use of refs, atoms, agents, and avoiding global state mutation in local scopes.").Severity(rules.SeverityWarning).
		When(func(_ *reader.RichNode, _ map[string]interface{}, filepath string) bool {
			// Skip REPL dev files (user.clj, dev/) where alter-var-root and component state resets are standard
			return !(strings.HasSuffix(filepath, "user.clj") || strings.Contains(filepath, "/dev/"))
		}).
		When(rules.IsList()).
		When(rules.HasMinChildren(1)).
		When(rules.ChildIsSymbol(0)).
		When(rules.Any(
			// 1. Chamada direta de função de mutação
			rules.Any(
				rules.ChildValueEquals(0, "set!"),
				rules.ChildValueEquals(0, "alter-var-root"),
				rules.ChildValueEquals(0, "agent-send"),
				rules.ChildValueEquals(0, "agent-send-off"),
				rules.ChildValueEquals(0, "intern"),
				rules.ChildValueEquals(0, "aset"),
			),
			// 2. def ou defonce dentro de escopo local
			rules.All(
				rules.Any(rules.ChildValueEquals(0, "def"), rules.ChildValueEquals(0, "defonce")),
				rules.IsLocalScope(),
			),
			// 3. ref-set fora de dosync
			rules.All(
				rules.ChildValueEquals(0, "ref-set"),
				rules.Not(rules.IsInside("dosync")),
			),
			// 4. Chamada de reset!
			rules.ChildValueEquals(0, "reset!"),
			// 5. Chamada de send ou send-off passando uma função com efeitos colaterais
			rules.All(
				rules.Any(rules.ChildValueEquals(0, "send"), rules.ChildValueEquals(0, "send-off")),
				rules.HasMinChildren(3),
				rules.ChildMatches(2, rules.Any(
					isSideEffectCall,
					rules.HasDescendant(isSideEffectCall),
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
		SeverityFunc(func(node *reader.RichNode, _ map[string]interface{}, defaultSev rules.Severity) rules.Severity {
			if len(node.Children) > 0 && node.Children[0].Value == "reset!" {
				return rules.SeverityInfo
			}
			return defaultSev
		}).
		Register()
}