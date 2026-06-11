package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

func init() {
	NewRule("single-segment-namespace").
		Name("Single-segment namespace").
		Description("Detects namespaces declared with a single segment (ns foo) instead of qualified names (ns my-app.foo).").
		Severity(SeverityWarning).
		When(IsList()).
		When(HasMinChildren(2)).
		When(ChildValueEquals(0, "ns")).
		When(ChildIsSymbol(1)).
		When(func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
			return !strings.Contains(node.Children[1].Value, ".")
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
