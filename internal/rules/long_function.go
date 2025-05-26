// Package rules implementa regras para detectar funções excessivamente longas em Clojure
// Esta regra específica identifica funções que excedem um limite de linhas significativas
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// LongFunctionRule detecta funções que são muito longas
// Funções longas são difíceis de entender, testar e manter
type LongFunctionRule struct {
	Rule
	MaxLines int `json:"max_lines" yaml:"max_lines"` // Limite máximo de linhas significativas
}

func (r *LongFunctionRule) Meta() Rule {
	return r.Rule
}

// countSignificantLines conta linhas significativas em um nó, excluindo comentários e newlines
// Considera bindings em let como linhas adicionais pois aumentam complexidade
func countSignificantLines(node *reader.RichNode) int {
	if node == nil {
		return 0
	}

	count := 0
	// Conta apenas nós significativos (não comentários ou newlines)
	if node.Type != reader.NodeComment && node.Type != reader.NodeNewline {
		count = 1

		// Tratamento especial para let: cada binding conta como linha adicional
		if node.Type == reader.NodeList && len(node.Children) > 0 &&
			node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "let" {

			if len(node.Children) > 1 && node.Children[1].Type == reader.NodeVector {
				// Cada par binding-valor conta como uma linha
				count += len(node.Children[1].Children) / 2
			}
		}
	}

	// Conta recursivamente nos filhos
	for _, child := range node.Children {
		count += countSignificantLines(child)
	}

	return count
}

// Check analisa definições de função procurando por funções muito longas
// Usa contagem de linhas significativas incluindo bindings let
func (r *LongFunctionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Verifica se é uma definição de função (defn)
	if node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "defn" {

		// Extrai nome da função para mensagem informativa
		funcName := "unknown-function"
		if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
			funcName = node.Children[1].Value
		}

		// Conta linhas significativas na função
		significantLines := countSignificantLines(node)

		// Verifica se excede o limite configurado
		if significantLines > r.MaxLines {
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Function %q is too long: %d significant lines (max %d). Consider breaking it into smaller functions. Each binding in let blocks counts as a significant line.", funcName, significantLines, r.MaxLines),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}
	return nil
}

// init registra a regra de função longa com configurações padrão
// Limite de 50 linhas balanceia flexibilidade com manutenibilidade
func init() {
	defaultRule := &LongFunctionRule{
		Rule: Rule{
			ID:          "long-function",
			Name:        "Long Function",
			Description: "Functions should be kept short and focused. Long functions are harder to understand, test, and maintain. Each binding in let blocks counts as a significant line.",
			Severity:    SeverityWarning,
		},
		MaxLines: 50, // Limite padrão de 50 linhas significativas
	}

	RegisterRule(defaultRule)
}
