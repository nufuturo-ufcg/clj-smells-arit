// Package rules implementa regras para detectar otimização prematura em código Clojure
// Esta regra específica identifica padrões que sugerem complexidade desnecessária antes de profiling
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// Constantes para identificação da regra de otimização prematura
const (
	PrematureOptimizationRuleID   = "premature-optimization"
	PrematureOptimizationRuleName = "Premature Optimization"
)

// PrematureOptimizationRule detecta padrões que sugerem otimização prematura
// Identifica código complexo introduzido sem evidência de necessidade de performance
type PrematureOptimizationRule struct{}

// NewPrematureOptimizationRule cria uma nova instância da regra
func NewPrematureOptimizationRule() RegisteredRule {
	return &PrematureOptimizationRule{}
}

func (r *PrematureOptimizationRule) Meta() Rule {
	return Rule{
		ID:       PrematureOptimizationRuleID,
		Name:     PrematureOptimizationRuleName,
		Severity: SeverityWarning,
		Description: "Identifies code patterns that suggest premature optimization, such as overly generic retry logic for I/O operations. " +
			"Optimizing code before identifying actual performance bottlenecks can lead to unnecessary complexity and reduced maintainability. " +
			"Consider if the optimization addresses a proven issue and if its complexity is justified.",
	}
}

// Check analisa nós procurando por padrões de otimização prematura
// Foca em identificar complexidade desnecessária como retry genérico
func (r *PrematureOptimizationRule) Check(node *reader.RichNode, context map[string]interface{}, filepathAlias string) *Finding {

	//fmt.Println("!!! PrematureOptimizationRule CHECK CALLED (conforming to CheckerRule) !!!")

	// Verifica se é uma lista (chamada de função)
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		firstChild := node.Children[0]

		// Detecta uso de 'with-retry' que pode indicar otimização prematura
		if firstChild.Type == reader.NodeSymbol && firstChild.Value == "with-retry" {

			//fmt.Printf("[DEBUG PrematureOpt - CheckerRule] Found with-retry: %s (Line: %d) in file %s\\n", node.Value, node.Location.StartLine, filepathAlias)
			meta := r.Meta()
			return &Finding{
				RuleID:   meta.ID,
				Severity: meta.Severity,
				Message:  fmt.Sprintf("Premature optimization: Generic 'with-retry' at line %d. Consider if this complexity is justified without profiling.", node.Location.StartLine),
				Location: node.Location,
				Filepath: filepathAlias,
			}
		}
	}

	return nil
}

// isIoCallRecursive verifica recursivamente se há chamadas de I/O
// Atualmente simplificado e desabilitado para evitar complexidade desnecessária
func isIoCallRecursive(node *reader.RichNode) bool {
	fmt.Println("[DEBUG isIoCallRecursive - SIMPLIFIED AND DISABLED] Called, will return false.")

	return false
}

// isDirectIoCall verifica se um nó representa uma chamada direta de I/O
// Identifica funções comuns de I/O que podem se beneficiar de retry
func isDirectIoCall(node *reader.RichNode) bool {
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		funcNode := node.Children[0]
		if funcNode.Type == reader.NodeSymbol {
			// Lista de funções de I/O conhecidas que podem falhar
			switch funcNode.Value {
			case "http/get", "clj-http.client/get", "slurp", "clojure.java.io/reader",
				"get-file-from-network-problematic", "get-another-file-from-network":
				return true
			}
		}
	}
	return false
}

// init registra a regra de otimização prematura
// Configurada como WARNING pois pode indicar problema arquitetural
func init() {

	RegisterRule(NewPrematureOptimizationRule())
	//fmt.Println("!!! PrematureOptimizationRule REGISTERED via init() !!!")
}
