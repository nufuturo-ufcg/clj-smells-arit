// Package rules implementa regras para detectar geradores potencialmente ineficientes em Clojure
// Esta regra específica identifica uso de gen/such-that que pode ser ineficiente
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// PotentiallyInefficientGeneratorRule detecta uso de geradores que podem ser ineficientes
// Especificamente identifica gen/such-that que pode gerar muitas tentativas desnecessárias
type PotentiallyInefficientGeneratorRule struct{}

func (r *PotentiallyInefficientGeneratorRule) Meta() Rule {
	return Rule{
		ID:          "inefficient-filter: inefficient-generator",
		Name:        "Potentially Inefficient Generator",
		Description: "Detects the use of generator functions like `gen/such-that` which might be inefficient if the predicate is rarely satisfied by the base generator, potentially leading to excessive generation attempts.",
		Severity:    SeverityHint,
	}
}

// Check analisa nós procurando por uso de gen/such-that
// Este gerador pode ser ineficiente se o predicado for raramente satisfeito
func (r *PotentiallyInefficientGeneratorRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	// Deve ser uma lista com pelo menos 3 elementos (função, predicado, gerador)
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	funcNode := node.Children[0]

	// Verifica se é uma chamada para gen/such-that
	if funcNode.Type != reader.NodeSymbol || funcNode.Value != "gen/such-that" {
		return nil
	}

	// gen/such-that detectado - pode ser ineficiente dependendo do predicado
	meta := r.Meta()
	return &Finding{
		RuleID:   meta.ID,
		Message:  fmt.Sprintf("Using `gen/such-that` can be inefficient if the predicate is rarely satisfied by the base generator. Review if this usage might lead to excessive generation attempts."),
		Filepath: filepath,
		Location: node.Location,
		Severity: meta.Severity,
	}
}

// init registra a regra de gerador potencialmente ineficiente
// Configurada como HINT pois é uma sugestão de revisão de performance
func init() {
	RegisterRule(&PotentiallyInefficientGeneratorRule{})
}
