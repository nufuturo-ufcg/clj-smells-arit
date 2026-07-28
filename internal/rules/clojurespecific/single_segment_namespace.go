package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

func init() {
	rules.NewRule("single-segment-namespace").
		Name("Single-segment namespace").
		Description("Detects namespaces declared with a single segment (ns foo) instead of qualified names (ns my-app.foo).").Severity(rules.SeverityWarning).
		When(rules.IsList()).
		When(rules.HasMinChildren(2)).
		When(rules.ChildValueEquals(0, "ns")).
		When(rules.ChildIsSymbol(1)).
		When(func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
			nsName := node.Children[1].Value
			return nsName != "user" && !strings.Contains(nsName, ".")
		}).
		MessageFunc(func(node *reader.RichNode, _ map[string]interface{}) string {
			name := node.Children[1].Value
			return fmt.Sprintf(
				"Single-segment namespace '%s' detected. Prefer qualified namespaces (e.g. my-app.%s) to avoid collisions and tooling issues.",
				name,
				name,
			)
		}).
		Register()
}