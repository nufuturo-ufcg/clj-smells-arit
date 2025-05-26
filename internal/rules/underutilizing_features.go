// Package rules implementa regras para detectar subutilização de recursos do Clojure
// Esta regra específica identifica padrões que podem ser simplificados usando funções idiomáticas
package rules

import (
	"github.com/thlaurentino/arit/internal/reader"
)

// UseMapcatRule detecta uso de (apply concat (map ...)) que pode ser substituído por mapcat
// Promove código mais idiomático e potencialmente mais eficiente
type UseMapcatRule struct {
}

func (r *UseMapcatRule) Meta() Rule {
	return Rule{
		ID:          "underutilizing-features: use-mapcat",
		Name:        "Underutilizing features: Use mapcat",
		Description: "Detects usage of (apply concat (map ...)) which can be replaced by (mapcat ...)",
		Severity:    SeverityHint,
	}
}

// Check analisa nós procurando pelo padrão (apply concat (map ...))
// Este padrão é comum mas pode ser simplificado usando mapcat
func (r *UseMapcatRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	// Deve ser uma lista (chamada de função)
	if node.Type != reader.NodeList {
		return nil
	}

	// Deve ter exatamente 3 elementos: apply, concat, e (map ...)
	if len(node.Children) != 3 {
		return nil
	}

	applyNode := node.Children[0]
	concatNode := node.Children[1]
	mapFormNode := node.Children[2]

	// Primeiro elemento deve ser 'apply'
	if applyNode.Type != reader.NodeSymbol || applyNode.Value != "apply" {
		return nil
	}

	// Segundo elemento deve ser 'concat'
	if concatNode.Type != reader.NodeSymbol || concatNode.Value != "concat" {
		return nil
	}

	// Terceiro elemento deve ser uma forma (map ...)
	if mapFormNode.Type != reader.NodeList {
		return nil
	}

	// A forma deve começar com 'map' e ter pelo menos uma função e uma coleção
	if len(mapFormNode.Children) < 2 || mapFormNode.Children[0].Type != reader.NodeSymbol || mapFormNode.Children[0].Value != "map" {
		return nil
	}

	// Padrão detectado: (apply concat (map ...)) pode ser substituído por mapcat
	meta := r.Meta()
	finding := &Finding{
		RuleID:   meta.ID,
		Message:  "Consider using `mapcat` instead of `(apply concat (map ...))` for better performance and idiomatic style.",
		Filepath: filepath,
		Location: node.Location,
		Severity: meta.Severity,
	}
	return finding
}

// init registra a regra de uso de mapcat
// Configurada como HINT pois é uma sugestão de melhoria, não um erro
func init() {

	defaultRule := &UseMapcatRule{}

	RegisterRule(defaultRule)
}
