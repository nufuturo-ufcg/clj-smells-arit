package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

// nonIdiomaticParameterBindingRule detecta o padrão & [x] para capturar
// um único parâmetro opcional em funções. Esse padrão é não-idiomático em Clojure:
// força o chamador a lidar com aridade variádica quando a intenção é apenas
// um parâmetro opcional. A forma idiomática é usar múltiplas aridades de função
// ou um mapa de opções.
//
// Detecta:
//   (defn f [x & [y]] ...)       → smell: & [y] com apenas 1 elemento no vetor rest
//   (defn f [x & [y z]] ...)     → smell: & [y z] — deveria usar mapa de opções
//   (fn [x & [y]] ...)           → smell: lambdas também
//   (defn- f [x & [y]] ...)      → smell: funções privadas também
//
// NÃO detecta:
//   (defn f [x & args] ...)      → legítimo: captura variádica aberta
//   (defmacro m [& body] ...)    → legítimo: macros com body variádico
type nonIdiomaticParameterBindingRule struct {
	rules.Rule
}

func (r *nonIdiomaticParameterBindingRule) Meta() rules.Rule {
	return r.Rule
}

// findParamVector retorna o vetor de parâmetros de um defn/fn/defn-.
// A estrutura é: (defn name docstring? [params] body)
// ou multi-arity: (defn name docstring? ([params] body) ...)
func findParamVector(node *reader.RichNode) *reader.RichNode {
	if len(node.Children) < 3 {
		return nil
	}
	for i := 2; i < len(node.Children); i++ {
		child := node.Children[i]
		if child.Type == reader.NodeVector {
			return child
		}
		// Pula docstring
		if child.Type == reader.NodeString {
			continue
		}
		// Pula metadados (maps de metadados)
		if child.Type == reader.NodeMap {
			continue
		}
		// Se encontrou algo que não é vetor nem string/map, para
		break
	}
	return nil
}

// hasRestDestructuring verifica se um vetor de parâmetros tem & [x ...] (rest destructuring)
// e retorna (true, contagem de elementos no vetor de rest) se encontrar.
func hasRestDestructuring(paramVec *reader.RichNode) (bool, int, string) {
	if paramVec == nil || paramVec.Type != reader.NodeVector {
		return false, 0, ""
	}
	children := paramVec.Children
	for i := 0; i < len(children)-1; i++ {
		// Procura pelo símbolo "&"
		if children[i].Type == reader.NodeSymbol && children[i].Value == "&" {
			next := children[i+1]
			// O padrão problemático: & [x] ou & [x y z] — rest como vetor destruturado
			if next.Type == reader.NodeVector {
				elemCount := 0
				names := []string{}
				for _, el := range next.Children {
					if el.Type == reader.NodeSymbol && el.Value != "&" {
						elemCount++
						names = append(names, el.Value)
					}
				}
				if elemCount > 0 {
					descr := "["
					for j, n := range names {
						if j > 0 {
							descr += " "
						}
						descr += n
					}
					descr += "]"
					return true, elemCount, descr
				}
			}
		}
	}
	return false, 0, ""
}

func (r *nonIdiomaticParameterBindingRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	first := node.Children[0]
	if first.Type != reader.NodeSymbol {
		return nil
	}

	sym := first.Value

	// Só verifica defn, defn- e fn (não defmacro — rest args em macros são idiomáticos)
	if sym != "defn" && sym != "defn-" && sym != "fn" {
		return nil
	}

	// Busca o vetor de parâmetros
	paramVec := findParamVector(node)
	if paramVec == nil {
		return nil
	}

	hasRest, elemCount, restDescr := hasRestDestructuring(paramVec)
	if !hasRest {
		return nil
	}

	// Determina a mensagem de refatoração baseado na quantidade de parâmetros
	var suggestion string
	if elemCount == 1 {
		suggestion = "use multiple arities: ([x] (f x default)) ([x opt] ...)"
	} else {
		suggestion = "use an options map: ([x {:keys [" + restDescr[1:len(restDescr)-1] + "] :or {...}}] ...)"
	}

	fnName := ""
	if sym != "fn" && len(node.Children) >= 2 && node.Children[1].Type == reader.NodeSymbol {
		fnName = " '" + node.Children[1].Value + "'"
	}

	return &rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Non-idiomatic parameter binding: function%s uses `& %s` for optional params. "+
				"This hides the real arity and confuses callers. Instead, %s.",
			fnName, restDescr, suggestion,
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func init() {
	rules.RegisterRule(&nonIdiomaticParameterBindingRule{
		Rule: rules.Rule{
			ID:          "non-idiomatic-parameter-binding",
			Name:        "Non-Idiomatic Parameter Binding",
			Description: "Detects functions using & [x] destructuring for optional parameters. This pattern hides the real arity and is non-idiomatic. Prefer multiple arities or an options map.",
			Severity:    rules.SeverityInfo,
		},
	})
}
