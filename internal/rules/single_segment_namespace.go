package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type SingleSegmentNamespaceRule struct {
	Rule
}

func (r *SingleSegmentNamespaceRule) Meta() Rule {
	return r.Rule
}

func (r *SingleSegmentNamespaceRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	head := node.Children[0]
	if head.Type != reader.NodeSymbol || head.Value != "ns" {
		return nil
	}

	nsSym := node.Children[1]
	if nsSym.Type != reader.NodeSymbol || nsSym.Value == "" {
		return nil
	}

	name := nsSym.Value
	if hasDot(name) {
		return nil
	}

	msg := fmt.Sprintf(
		"Single-segment namespace '%s' detected. Prefer qualified namespaces (e.g. my-app.%s) to avoid collisions and tooling issues.",
		name,
		name,
	)

	return &Finding{
		RuleID:   r.ID,
		Message:  msg,
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func hasDot(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return true
		}
	}
	return false
}

func init() {
	defaultRule := &SingleSegmentNamespaceRule{
		Rule: Rule{
			ID:          "single-segment-namespace",
			Name:        "Single-segment namespace",
			Description: "Detects namespaces declared with a single segment (ns foo) instead of qualified names (ns my-app.foo).",
			Severity:    SeverityWarning,
		},
	}
	RegisterRule(defaultRule)
}
