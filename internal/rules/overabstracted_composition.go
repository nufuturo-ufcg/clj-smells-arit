// Package rules implementa regras para detectar composições excessivamente abstratas em Clojure
// Esta regra específica identifica uso excessivo da função comp que pode prejudicar legibilidade
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// compositionFunctionArgsThreshold define o limite máximo recomendado de funções em uma composição
// Acima deste limite, a composição pode se tornar difícil de entender e debugar
const compositionFunctionArgsThreshold = 3

// OverabstractedCompositionRule detecta composições com muitas funções usando comp
// Composições longas podem prejudicar legibilidade e dificultam debugging
type OverabstractedCompositionRule struct{}

func (r *OverabstractedCompositionRule) Meta() Rule {

	//const compositionFunctionArgsThreshold = 3
	return Rule{
		ID:          "overabstracted-composition",
		Name:        "Overabstracted Composition",
		Description: fmt.Sprintf("Detects compositions using `comp` that involve more than %d functions. Long chains of composition can sometimes hinder readability and debugging. Consider breaking down complex compositions.", compositionFunctionArgsThreshold),
		Severity:    SeverityHint,
	}
}

// Check analisa nós procurando por usos da função comp com muitos argumentos
// Sugere refatoração quando o número de funções excede o limite recomendado
func (r *OverabstractedCompositionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	//const compositionFunctionArgsThreshold = 3

	// Deve ser uma lista (chamada de função)
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	// Primeiro elemento deve ser o símbolo 'comp'
	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol || firstElement.Value != "comp" {
		return nil
	}

	// Conta o número de funções sendo compostas (exclui o símbolo 'comp')
	numberOfFunctions := len(node.Children) - 1

	// Verifica se excede o limite recomendado
	if numberOfFunctions > compositionFunctionArgsThreshold {
		meta := r.Meta()
		message := fmt.Sprintf("The `comp` form at this location is composing %d functions, which exceeds the recommended maximum of %d. This can make the code harder to understand and debug. Consider refactoring into smaller, named compositions or using a threading macro if appropriate.", numberOfFunctions, compositionFunctionArgsThreshold)

		return &Finding{
			RuleID:   meta.ID,
			Message:  message,
			Filepath: filepath,
			Location: node.Location,
			Severity: meta.Severity,
		}
	}

	return nil
}

// init registra a regra de composição excessivamente abstrata
// Configurada como HINT pois é uma sugestão de melhoria de legibilidade
func init() {
	RegisterRule(&OverabstractedCompositionRule{})
}
