// Package rules implementa regras para detectar listas de parâmetros excessivamente longas em Clojure
// Esta regra específica identifica funções com muitos parâmetros que dificultam uso e manutenção
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// LongParameterListRule detecta funções com muitos parâmetros
// Muitos parâmetros tornam funções difíceis de usar e lembrar
type LongParameterListRule struct {
	Rule
	MaxParameters int `json:"max_parameters" yaml:"max_parameters"` // Limite máximo de parâmetros
}

func (r *LongParameterListRule) Meta() Rule {
	return r.Rule
}

// Check analisa definições de função procurando por listas longas de parâmetros
// Sugere uso de mapas para agrupar parâmetros relacionados
func (r *LongParameterListRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	// Verifica se é uma definição de função (defn)
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "defn" {
		// Extrai nome da função para mensagem informativa
		funcName := "unknown-function"
		if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
			funcName = node.Children[1].Value
		}

		// Localiza o vetor de argumentos, considerando docstring opcional
		var argsNode *reader.RichNode
		argsNodeIndex := 2
		if len(node.Children) > argsNodeIndex && node.Children[argsNodeIndex].Type == reader.NodeString {
			// Pula docstring se presente
			argsNodeIndex = 3
		}

		if len(node.Children) > argsNodeIndex {
			argsNode = node.Children[argsNodeIndex]
		} else {
			return nil
		}

		// Verifica se o vetor de argumentos existe e conta parâmetros
		if argsNode != nil && argsNode.Type == reader.NodeVector {
			paramCount := len(argsNode.Children)
			if paramCount > r.MaxParameters {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Function %q has too many parameters: %d (max %d). Consider using a map.", funcName, paramCount, r.MaxParameters),
					Filepath: filepath,
					Location: argsNode.Location,
					Severity: r.Severity,
				}
			}
		} else {
			// Vetor de argumentos não encontrado ou malformado
		}
	}
	return nil
}

// init registra a regra de lista longa de parâmetros com configurações padrão
// Limite de 5 parâmetros balanceia flexibilidade com usabilidade
func init() {

	defaultRule := &LongParameterListRule{
		Rule: Rule{
			ID:          "long-parameter-list",
			Name:        "Long Parameter List",
			Description: "Functions should not have an excessive number of parameters. Consider grouping related parameters into a map or record.",
			Severity:    SeverityWarning,
		},
		MaxParameters: 5, // Limite padrão de 5 parâmetros
	}

	RegisterRule(defaultRule)
}
