// Package rules implementa regras para detectar lambdas desnecessárias em Clojure
// Esta regra específica identifica lambdas triviais que apenas delegam para outras funções
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// TrivialLambdaRule detecta lambdas que apenas chamam outra função com os mesmos argumentos
// Estes lambdas são desnecessários e podem ser substituídos pela função diretamente
type TrivialLambdaRule struct {
	Rule
}

func (r *TrivialLambdaRule) Meta() Rule {
	return r.Rule
}

// Check analisa nós procurando por lambdas triviais
// Verifica tanto function literals (#(...)) quanto formas fn explícitas
func (r *TrivialLambdaRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	var lambdaArgs []*reader.RichNode
	var lambdaBody []*reader.RichNode
	isFnLiteral := false

	// Analisa formas fn explícitas: (fn [args] body) ou (fn name [args] body)
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "fn" {

		// Determina índices dos argumentos e corpo, considerando nome opcional
		argsIndex := 1
		bodyIndex := 2
		if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
			// Função nomeada: (fn name [args] body)
			argsIndex = 2
			bodyIndex = 3
		}
		// Extrai argumentos e corpo se presentes
		if len(node.Children) > argsIndex && node.Children[argsIndex].Type == reader.NodeVector {
			lambdaArgs = node.Children[argsIndex].Children
			if len(node.Children) > bodyIndex {
				lambdaBody = node.Children[bodyIndex:]
			}
		}
	} else if node.Type == reader.NodeFnLiteral {
		// Analisa function literals: #(...)
		isFnLiteral = true
		lambdaBody = node.Children
	}

	// Processa o corpo do lambda se disponível
	if lambdaBody != nil {

		// Extrai a chamada interna dependendo do tipo de lambda
		var innerCallNodes []*reader.RichNode
		if !isFnLiteral {
			// Para fn explícito, o corpo deve ser uma única chamada
			if len(lambdaBody) == 1 && lambdaBody[0].Type == reader.NodeList {
				innerCallNodes = lambdaBody[0].Children
			}
		} else {
			// Para function literal, o corpo é a chamada direta
			innerCallNodes = lambdaBody
		}

		// Verifica se há uma chamada de função válida
		if len(innerCallNodes) > 0 && innerCallNodes[0].Type == reader.NodeSymbol {
			calledFuncSymbol := innerCallNodes[0]
			innerArgs := innerCallNodes[1:]

			argsMatch := false
			if isFnLiteral {
				// Verifica correspondência de argumentos para function literals
				if len(innerArgs) == 0 {
					// #(func) - sem argumentos
					argsMatch = true
				} else if len(innerArgs) == 1 && innerArgs[0].Type == reader.NodeSymbol && innerArgs[0].Value == "%" {
					// #(func %) - um argumento
					argsMatch = true
				} else {
					// #(func %1 %2 %3) - múltiplos argumentos numerados
					allArgsAreNumbered := true
					maxN := 0
					for i, arg := range innerArgs {
						if arg.Type == reader.NodeSymbol && len(arg.Value) > 1 && arg.Value[0] == '%' {
							num := 0
							_, err := fmt.Sscan(arg.Value[1:], &num)
							if err != nil || num != i+1 {
								allArgsAreNumbered = false
								break
							}
							if num > maxN {
								maxN = num
							}
						} else {
							allArgsAreNumbered = false
							break
						}
					}

					// Argumentos devem estar em sequência (%1 %2 %3...)
					if allArgsAreNumbered && len(innerArgs) == maxN {
						argsMatch = true
					}
				}
			} else {
				// Verifica correspondência de argumentos para fn explícito
				if len(lambdaArgs) == len(innerArgs) {
					match := true
					for i := range lambdaArgs {
						// Argumentos devem ter nomes idênticos na mesma ordem
						if !(lambdaArgs[i].Type == reader.NodeSymbol && innerArgs[i].Type == reader.NodeSymbol && lambdaArgs[i].Value == innerArgs[i].Value) {
							match = false
							break
						}
					}
					argsMatch = match
				}
			}

			// Se os argumentos correspondem, é um lambda trivial
			if argsMatch {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Trivial lambda/fn. Consider using function %q directly.", calledFuncSymbol.Value),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}
	return nil
}

// init registra a regra de lambda trivial com configurações padrão
// Configurada como INFO pois é uma questão de simplicidade de código
func init() {

	defaultRule := &TrivialLambdaRule{
		Rule: Rule{
			ID:          "trivial-lambda",
			Name:        "Trivial Lambda",
			Description: "Lambda or fn that merely calls another function with the same arguments can be replaced by the function itself.",
			Severity:    SeverityInfo,
		},
	}

	RegisterRule(defaultRule)
}
