package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type SingleSegmentNamespaceRule struct {
	rules.Rule
}

func (r *SingleSegmentNamespaceRule) Meta() rules.Rule {
	return r.Rule
}

func (r *SingleSegmentNamespaceRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	// 1. Context Awareness: Ignora ambientes onde single-segment namespaces são seguros/esperados
	lowerPath := strings.ToLower(filepath)
	if strings.Contains(lowerPath, "/test/") ||
		strings.Contains(lowerPath, "/scripts/") ||
		strings.Contains(lowerPath, "/dev/") ||
		strings.Contains(lowerPath, "/build/") ||
		strings.Contains(lowerPath, "/support/") ||
		strings.Contains(lowerPath, "project.clj") {
		return nil
	}

	if node.Type == reader.NodeList && len(node.Children) >= 2 {
		if node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "ns" {
			if node.Children[1].Type == reader.NodeSymbol {
				nsName := node.Children[1].Value

				// 2. Safelist Estendida: Perdoa namespaces de segmento único que são padrão no ecossistema
				switch nsName {
				case "user", "dev", "test", "build", "repl", "script", "scratch":
					return nil
				}

				// 3. Condição Principal: Falta de ponto qualificador
				if !strings.Contains(nsName, ".") {
					return &rules.Finding{
						RuleID:   r.ID,
						Message:  fmt.Sprintf("Single-segment namespace '%s' detected. Prefer qualified namespaces (e.g. my-app.%s) to avoid collisions and tooling issues.", nsName, nsName),
						Filepath: filepath,
						Location: node.Location,
						Severity: r.Severity,
					}
				}
			}
		}
	}

	return nil
}

func init() {
	rules.RegisterRule(&SingleSegmentNamespaceRule{
		Rule: rules.Rule{
			ID:          "single-segment-namespace",
			Name:        "Single-segment namespace",
			Description: "Detects namespaces declared with a single segment (ns foo) instead of qualified names (ns my-app.foo).",
			Severity:    rules.SeverityWarning,
		},
	})
}