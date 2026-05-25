package rules

import (
	"fmt"

	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type UnmanagedResourceIORule struct {
	Rule
}

func (r *UnmanagedResourceIORule) Meta() Rule {
	return r.Rule
}

func (r *UnmanagedResourceIORule) checkIoOperations(symbol string) bool {
	alvos := []string{"reader", "writer", "stream"}

	if strings.Contains(symbol, "io") {
		for _, opcao := range alvos {
			if strings.Contains(strings.ToLower(symbol), opcao) {
				return true
			}
		}
	}
	return false
}

func (r *UnmanagedResourceIORule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) <= 0 || node.Children[0].Type != reader.NodeSymbol {
		return nil
	}

	val, ok := context["isInsideWithOpen"].(bool)

	if r.checkIoOperations(node.Children[0].Value) && ok && !val {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("I/O resource used without with-open: use with-open to ensure the resource is closed."),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}
	return nil
}

func init() {
	defaultRule := &UnmanagedResourceIORule{
		Rule: Rule{
			ID:          "unmanaged-resource_io",
			Name:        "Unmanaged Resource I/O",
			Description: "Detects I/O resources (reader, writer, stream) used without with-open, which can cause resource leaks.",
			Severity:    SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
}
