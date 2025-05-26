// Package rules implementa regras de análise para detecção de code smells em Clojure
// Esta regra específica detecta funções "Middle Man" que apenas delegam para outras funções
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// MiddleManRule detecta funções que servem apenas como intermediárias
// Identifica funções que simplesmente delegam chamadas para outras funções sem agregar valor
type MiddleManRule struct {
	Rule
}

func (r *MiddleManRule) Meta() Rule {
	return r.Rule
}

// Check analisa nós de definição de função procurando por padrões de Middle Man
// Uma função Middle Man é aquela que apenas repassa parâmetros para outra função
func (r *MiddleManRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	// Verifica se é uma definição de função (defn)
	if node.Type == reader.NodeList && len(node.Children) > 1 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "defn" {

		funcNameNode := node.Children[1]

		// Localiza o vetor de parâmetros, considerando docstring opcional
		paramsNodeIndex := 2
		bodyStartIndex := 3
		if len(node.Children) > paramsNodeIndex && node.Children[paramsNodeIndex].Type == reader.NodeString {
			// Pula docstring se presente
			paramsNodeIndex = 3
			bodyStartIndex = 4
		}

		// Verifica se há vetor de parâmetros válido
		if len(node.Children) <= paramsNodeIndex || node.Children[paramsNodeIndex].Type != reader.NodeVector {
			return nil
		}
		paramsNode := node.Children[paramsNodeIndex]
		outerParams := paramsNode.Children

		// Verifica se há corpo da função
		if len(node.Children) <= bodyStartIndex {
			return nil
		}
		bodyNodes := node.Children[bodyStartIndex:]

		// Encontra o único nó significativo no corpo (ignora comentários e newlines)
		var significantBodyNode *reader.RichNode
		for _, bNode := range bodyNodes {
			if bNode.Type != reader.NodeComment && bNode.Type != reader.NodeNewline {
				if significantBodyNode != nil {
					// Mais de um nó significativo - não é Middle Man simples
					significantBodyNode = nil
					break
				}
				significantBodyNode = bNode
			}
		}

		// Deve ter exatamente um nó significativo no corpo
		if significantBodyNode == nil {
			return nil
		}

		// O nó significativo deve ser uma chamada de função (lista)
		if significantBodyNode.Type != reader.NodeList || len(significantBodyNode.Children) == 0 {
			return nil
		}

		// Analisa a chamada interna
		innerCall := significantBodyNode.Children
		innerFuncNameNode := innerCall[0]
		innerArgs := innerCall[1:]

		// A função chamada deve ser um símbolo
		if innerFuncNameNode.Type != reader.NodeSymbol {
			return nil
		}

		numOuter := len(outerParams)
		numInner := len(innerArgs)

		// Verifica se é um middle man: função que apenas delega para outra função
		// com os mesmos parâmetros ou um subconjunto deles em ordem
		if numOuter > 0 && numInner >= numOuter {
			match := r.checkParameterMatch(outerParams, innerArgs, numOuter, numInner)

			if match {
				// Constrói mensagem informativa sobre o Middle Man detectado
				funcName := "?"
				if funcNameNode.Type == reader.NodeSymbol {
					funcName = funcNameNode.Value
				}
				innerFuncName := innerFuncNameNode.Value
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Function %q appears to be a 'Middle Man' delegating directly to %q. Consider using %q directly.", funcName, innerFuncName, innerFuncName),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}
	return nil
}

// checkParameterMatch verifica se os parâmetros externos correspondem aos argumentos internos
// Implementa lógica para detectar delegação direta de parâmetros
func (r *MiddleManRule) checkParameterMatch(outerParams, innerArgs []*reader.RichNode, numOuter, numInner int) bool {
	// Verifica se os últimos numOuter argumentos internos correspondem aos parâmetros externos
	// Permite argumentos extras no início (currying ou configuração)
	for i := 0; i < numOuter; i++ {
		outerParam := outerParams[i]
		innerArgIndex := numInner - numOuter + i
		innerArg := innerArgs[innerArgIndex]

		if !r.symbolsMatch(outerParam, innerArg) {
			return false
		}
	}

	// Se há argumentos extras no início (numInner > numOuter), verifica se eles não são
	// parâmetros da função externa (para evitar falsos positivos)
	if numInner > numOuter {
		prefixLen := numInner - numOuter
		for k := 0; k < prefixLen; k++ {
			prefixArg := innerArgs[k]
			if prefixArg.Type == reader.NodeSymbol {
				// Verifica se este argumento prefixo é um dos parâmetros externos
				for _, outerParam := range outerParams {
					if r.symbolsMatch(prefixArg, outerParam) {
						return false // Falso positivo: parâmetro sendo usado fora de ordem
					}
				}
			}
		}
	}

	return true
}

// symbolsMatch verifica se dois símbolos correspondem, usando ResolvedDefinition quando disponível
// Implementa comparação robusta de símbolos considerando resolução de escopo
func (r *MiddleManRule) symbolsMatch(sym1, sym2 *reader.RichNode) bool {
	if sym1.Type != reader.NodeSymbol || sym2.Type != reader.NodeSymbol {
		return false
	}

	// Primeiro, tenta usar ResolvedDefinition se ambos estiverem disponíveis
	// Esta é a comparação mais precisa pois considera resolução de escopo
	if sym1.ResolvedDefinition != nil && sym2.ResolvedDefinition != nil {
		return sym1.ResolvedDefinition == sym2.ResolvedDefinition
	}

	// Fallback: compara nomes dos símbolos
	// Útil quando ResolvedDefinition não está disponível (como com parâmetros)
	return sym1.Value == sym2.Value && sym1.Value != "" && sym2.Value != ""
}

// init registra a regra Middle Man com configurações padrão
// Configurada como HINT pois nem sempre indica problema real
func init() {
	RegisterRule(&MiddleManRule{
		Rule: Rule{
			ID:          "middle-man",
			Name:        "Middle Man Function",
			Description: "Identifies functions that primarily delegate calls to other functions (often HOFs like map/filter) without adding significant logic, increasing unnecessary indirection.",
			Severity:    SeverityHint,
		},
	})

}
