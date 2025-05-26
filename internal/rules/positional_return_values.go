package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// PositionalReturnValuesRule detecta funções que retornam coleções sequenciais
// onde o significado dos elementos é implícito por sua posição.
type PositionalReturnValuesRule struct {
	Rule
}

// Meta retorna os metadados da regra.
func (r *PositionalReturnValuesRule) Meta() Rule {
	return r.Rule
}

// isLiteralVector verifica se o nó é um vetor literal com múltiplos elementos.
func isLiteralVector(node *reader.RichNode) bool {
	return node.Type == reader.NodeVector && len(node.Children) >= 2
}

// isLiteralList verifica se o nó é uma lista literal (não uma chamada de função) com múltiplos elementos.
// Uma lista literal tipicamente começa com um valor literal ao invés de um símbolo que poderia ser uma função.
func isLiteralList(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}

	firstChild := node.Children[0]
	return firstChild.Type == reader.NodeNumber ||
		firstChild.Type == reader.NodeString ||
		firstChild.Type == reader.NodeKeyword ||
		firstChild.Type == reader.NodeBool ||
		firstChild.Type == reader.NodeNil
}

// isPositionalCollection verifica se uma coleção contém múltiplos valores que poderiam ser posicionais.
func isPositionalCollection(node *reader.RichNode) bool {
	return (isLiteralVector(node) || isLiteralList(node)) && len(node.Children) >= 2
}

// findFunctionBody localiza o índice de início do corpo para definições de função.
func (r *PositionalReturnValuesRule) findFunctionBody(node *reader.RichNode, fnType string) int {
	if fnType == "defn" {
		if len(node.Children) < 3 {
			return -1
		}
		bodyStartIndex := 2
		for i := 2; i < len(node.Children); i++ {
			if node.Children[i].Type == reader.NodeVector {
				return i + 1
			}
			if i == len(node.Children)-1 {
				return -1
			}
		}
		return bodyStartIndex
	} else if fnType == "fn" {
		if len(node.Children) < 2 {
			return -1
		}
		for i := 1; i < len(node.Children); i++ {
			if node.Children[i].Type == reader.NodeVector {
				return i + 1
			}
			if i == len(node.Children)-1 {
				return -1
			}
		}
	}
	return -1
}

// Check executa a verificação da regra para valores de retorno posicionais.
func (r *PositionalReturnValuesRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || (firstChild.Value != "defn" && firstChild.Value != "fn") {
		return nil
	}

	bodyStartIndex := r.findFunctionBody(node, firstChild.Value)
	if bodyStartIndex == -1 || bodyStartIndex >= len(node.Children) {
		return nil
	}

	lastBodyForm := node.Children[len(node.Children)-1]

	if isPositionalCollection(lastBodyForm) {
		collectionType := "vetor"
		if lastBodyForm.Type == reader.NodeList {
			collectionType = "lista"
		}
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Função retorna um %s literal com múltiplos valores. Considere retornar um mapa com chaves descritivas ao invés de depender de valores posicionais.", collectionType),
			Filepath: filepath,
			Location: lastBodyForm.Location,
			Severity: r.Severity,
		}
	}

	// Verifica retornos posicionais dentro de expressões let
	if lastBodyForm.Type == reader.NodeList && len(lastBodyForm.Children) > 0 &&
		lastBodyForm.Children[0].Type == reader.NodeSymbol && lastBodyForm.Children[0].Value == "let" {
		if len(lastBodyForm.Children) >= 3 {
			lastExprInLet := lastBodyForm.Children[len(lastBodyForm.Children)-1]
			if isPositionalCollection(lastExprInLet) {
				collectionType := "vetor"
				if lastExprInLet.Type == reader.NodeList {
					collectionType = "lista"
				}
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Função retorna um %s literal com múltiplos valores via uma forma `let`. Considere retornar um mapa com chaves descritivas ao invés de valores posicionais.", collectionType),
					Filepath: filepath,
					Location: lastExprInLet.Location,
					Severity: r.Severity,
				}
			}
		}
	}

	return nil
}

func init() {
	RegisterRule(&PositionalReturnValuesRule{
		Rule: Rule{
			ID:          "positional-return-values",
			Name:        "Valores de Retorno Posicionais",
			Description: "Detecta funções que retornam coleções sequenciais (vetores ou listas) onde o significado dos elementos é implícito por sua posição. Recomenda retornar um mapa com chaves descritivas.",
			Severity:    SeverityWarning,
		},
	})
}
