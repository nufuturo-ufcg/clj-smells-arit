// Package rules implementa o sistema de regras para análise de código Clojure
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// Rule contém os metadados básicos de uma regra de análise
// Define identificação, nome, descrição e severidade padrão
type Rule struct {
	ID          string   `json:"id" yaml:"id"`                   // Identificador único da regra
	Name        string   `json:"name" yaml:"name"`               // Nome legível da regra
	Description string   `json:"description" yaml:"description"` // Descrição detalhada do que a regra verifica
	Severity    Severity `json:"severity" yaml:"severity"`       // Severidade padrão dos findings
}

// RegisteredRule interface básica para todas as regras registradas
// Fornece acesso aos metadados da regra
type RegisteredRule interface {
	Meta() Rule // Retorna os metadados da regra
}

// CheckerRule interface para regras que verificam nós da AST
// Estende RegisteredRule com funcionalidade de verificação
type CheckerRule interface {
	RegisteredRule

	// Check verifica um nó da AST e retorna um finding se houver problema
	// context contém informações adicionais como escopo e configuração
	Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding
}

// registry mantém todas as regras registradas no sistema
var registry = make(map[string]RegisteredRule)

// RegisterRule registra uma nova regra no sistema
// Panic se uma regra com o mesmo ID já estiver registrada
func RegisterRule(rule RegisteredRule) {
	id := rule.Meta().ID
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("rule: rule with ID %q already registered", id))
	}
	registry[id] = rule
}

// GetRule busca uma regra pelo ID
// Retorna a regra e um booleano indicando se foi encontrada
func GetRule(id string) (RegisteredRule, bool) {
	rule, exists := registry[id]
	return rule, exists
}

// AllRules retorna todas as regras registradas
// Útil para listagem e configuração dinâmica
func AllRules() []RegisteredRule {
	rules := make([]RegisteredRule, 0, len(registry))
	for _, rule := range registry {
		rules = append(rules, rule)
	}
	return rules
}
