package rules

import (
	"github.com/thlaurentino/arit/internal/reader"
)

type IsInsideRule struct {
	Rule
}

func (r *IsInsideRule) Meta() Rule {
	return Rule{
		ID:          "is-inside-test",
		Name:        "Is Inside Test",
		Description: "Regra de exemplo para testar a função genérica IsInside.",
		Severity:    SeverityInfo,
	}
}

func (r *IsInsideRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Vamos procurar por chamadas à função "println"
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		first := node.Children[0]
		if first.Type == reader.NodeSymbol && first.Value == "println" {

			// Usamos a nova função IsInside para ver se esse println
			// está dentro da "macro-nivel-2" (do nosso arquivo de teste)
			if r.IsInside(context, "macro-nivel-2") {
				return &Finding{
					RuleID:   r.Meta().ID,
					Message:  "Encontrou um 'println' aninhado dentro de 'macro-nivel-2'!",
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Meta().Severity,
				}
			}
		}
	}
	return nil
}

func init() {
	RegisterRule(&IsInsideRule{})
}
