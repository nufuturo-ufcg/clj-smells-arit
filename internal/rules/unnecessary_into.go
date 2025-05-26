// Package rules implementa regras para detectar uso desnecessário da função into em Clojure
// Esta regra específica identifica casos onde into é usado quando alternativas mais idiomáticas existem
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// UnnecessaryIntoRule detecta uso desnecessário da função into
// A função into é útil para combinar coleções, mas frequentemente é mal utilizada
// para transformações simples de tipo onde alternativas mais concisas existem
type UnnecessaryIntoRule struct {
	Rule
	CheckTypeTransformations bool `json:"check_type_transformations" yaml:"check_type_transformations"` // Verificar transformações de tipo
	CheckMapMapping          bool `json:"check_map_mapping" yaml:"check_map_mapping"`                   // Verificar mapeamento de mapas
	CheckTransducerAPI       bool `json:"check_transducer_api" yaml:"check_transducer_api"`             // Verificar API de transdutores
}

func (r *UnnecessaryIntoRule) Meta() Rule {
	return r.Rule
}

// typeTransformationPatterns define padrões de transformação de tipo que podem ser simplificados
var typeTransformationPatterns = map[string]string{
	"[]":  "vec", // (into [] xs) => (vec xs)
	"#{}": "set", // (into #{} xs) => (set xs)
	"()":  "seq", // (into () xs) => (seq xs) - menos comum mas possível
}

// isEmptyCollection verifica se um nó representa uma coleção vazia
func isEmptyCollection(node *reader.RichNode) (bool, string) {
	if node == nil {
		return false, ""
	}

	switch node.Type {
	case reader.NodeVector:
		if len(node.Children) == 0 {
			return true, "[]"
		}
	case reader.NodeSet:
		if len(node.Children) == 0 {
			return true, "#{}"
		}
	case reader.NodeList:
		if len(node.Children) == 0 {
			return true, "()"
		}
	}

	return false, ""
}

// isMapFunction verifica se uma função é uma função de mapeamento
func isMapFunction(funcName string) bool {
	mapFunctions := map[string]bool{
		"map":    true,
		"mapcat": true,
		"mapv":   true,
		"pmap":   true,
		"for":    true,
	}
	return mapFunctions[funcName]
}

// isFilterFunction verifica se uma função é uma função de filtragem
func isFilterFunction(funcName string) bool {
	filterFunctions := map[string]bool{
		"filter":       true,
		"remove":       true,
		"keep":         true,
		"keep-indexed": true,
		"distinct":     true,
		"take":         true,
		"drop":         true,
		"take-while":   true,
		"drop-while":   true,
	}
	return filterFunctions[funcName]
}

// checkTypeTransformation verifica padrões de transformação de tipo desnecessários
func (r *UnnecessaryIntoRule) checkTypeTransformation(node *reader.RichNode) *Finding {
	// Verifica padrão: (into [] xs) => (vec xs)
	if len(node.Children) == 3 {
		firstArg := node.Children[1]
		secondArg := node.Children[2]

		if isEmpty, collType := isEmptyCollection(firstArg); isEmpty {
			if replacement, exists := typeTransformationPatterns[collType]; exists {
				meta := r.Meta()
				return &Finding{
					RuleID: meta.ID,
					Message: fmt.Sprintf("Unnecessary use of 'into' for type transformation. Use '(%s %s)' instead of '(into %s %s)' for better readability and performance.",
						replacement, getNodeText(secondArg), collType, getNodeText(secondArg)),
					Filepath: "",
					Location: node.Location,
					Severity: meta.Severity,
				}
			}
		}
	}

	return nil
}

// checkMapMapping verifica padrões de mapeamento de mapas ineficientes
func (r *UnnecessaryIntoRule) checkMapMapping(node *reader.RichNode) *Finding {
	// Verifica padrão: (into {} (map (fn [[k v]] [k (f v)]) m))
	if len(node.Children) == 3 {
		firstArg := node.Children[1]
		secondArg := node.Children[2]

		// Verifica se o primeiro argumento é um mapa vazio
		if isEmpty, collType := isEmptyCollection(firstArg); isEmpty && collType == "{}" {
			// Verifica se o segundo argumento é uma chamada de map
			if secondArg.Type == reader.NodeList && len(secondArg.Children) > 0 {
				if funcNode := secondArg.Children[0]; funcNode.Type == reader.NodeSymbol {
					funcName := funcNode.Value
					if isMapFunction(funcName) {
						meta := r.Meta()
						return &Finding{
							RuleID:   meta.ID,
							Message:  fmt.Sprintf("Inefficient map mapping with 'into'. Consider using 'reduce-kv' for better performance when transforming map values: (reduce-kv (fn [m k v] (assoc m k (f v))) {} m)"),
							Filepath: "",
							Location: node.Location,
							Severity: meta.Severity,
						}
					}
				}
			}
		}
	}

	return nil
}

// checkTransducerAPI verifica se a API de transdutores pode ser usada
func (r *UnnecessaryIntoRule) checkTransducerAPI(node *reader.RichNode) *Finding {
	// Verifica padrão: (into coll (map f xs)) => (into coll (map f) xs)
	if len(node.Children) == 3 {
		firstArg := node.Children[1]
		secondArg := node.Children[2]

		// Verifica se o segundo argumento é uma chamada de função de transformação
		if secondArg.Type == reader.NodeList && len(secondArg.Children) >= 3 {
			if funcNode := secondArg.Children[0]; funcNode.Type == reader.NodeSymbol {
				funcName := funcNode.Value
				if isMapFunction(funcName) || isFilterFunction(funcName) {
					meta := r.Meta()
					return &Finding{
						RuleID: meta.ID,
						Message: fmt.Sprintf("Consider using transducer API for better performance. Use '(into %s (%s %s) %s)' instead of '(into %s (%s %s %s))' to leverage transducers.",
							getNodeText(firstArg), funcName, getNodeText(secondArg.Children[1]), getNodeText(secondArg.Children[2]),
							getNodeText(firstArg), funcName, getNodeText(secondArg.Children[1]), getNodeText(secondArg.Children[2])),
						Filepath: "",
						Location: node.Location,
						Severity: meta.Severity,
					}
				}
			}
		}
	}

	return nil
}

// getNodeText extrai o texto de um nó para exibição
func getNodeText(node *reader.RichNode) string {
	if node == nil {
		return "nil"
	}

	switch node.Type {
	case reader.NodeSymbol, reader.NodeKeyword, reader.NodeString, reader.NodeNumber:
		return node.Value
	case reader.NodeVector:
		return "[]"
	case reader.NodeSet:
		return "#{}"
	case reader.NodeMap:
		return "{}"
	case reader.NodeList:
		if len(node.Children) == 0 {
			return "()"
		}
		return "(...)"
	default:
		return "..."
	}
}

// Check analisa nós procurando por uso desnecessário da função into
func (r *UnnecessaryIntoRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Verifica se é uma chamada para a função 'into'
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol || funcNode.Value != "into" {
		return nil
	}

	// Verifica diferentes padrões de uso desnecessário

	// 1. Transformações de tipo (into [] xs) => (vec xs)
	if r.CheckTypeTransformations {
		if finding := r.checkTypeTransformation(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	// 2. Mapeamento de mapas ineficiente
	if r.CheckMapMapping {
		if finding := r.checkMapMapping(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	// 3. API de transdutores não utilizada
	if r.CheckTransducerAPI {
		if finding := r.checkTransducerAPI(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	return nil
}

// init registra a regra de Unnecessary Into com configurações padrão
func init() {
	defaultRule := &UnnecessaryIntoRule{
		Rule: Rule{
			ID:          "unnecessary-into",
			Name:        "Unnecessary Into",
			Description: "Detects unnecessary usage of the 'into' function when more idiomatic alternatives exist. The 'into' function is useful for combining collections, but is often misused for simple type transformations like (into [] coll) instead of (vec coll), or (into #{} coll) instead of (set coll). This rule also identifies inefficient map mapping patterns and missed opportunities to use the transducer API.",
			Severity:    SeverityHint,
		},
		CheckTypeTransformations: true, // Verificar transformações de tipo por padrão
		CheckMapMapping:          true, // Verificar mapeamento de mapas por padrão
		CheckTransducerAPI:       true, // Verificar API de transdutores por padrão
	}

	RegisterRule(defaultRule)
}
