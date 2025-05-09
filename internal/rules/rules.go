package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type Rule struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Severity    Severity `json:"severity" yaml:"severity"`
}

type RegisteredRule interface {
	Meta() Rule
}

type CheckerRule interface {
	RegisteredRule

	Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding
}

var registry = make(map[string]RegisteredRule)

func RegisterRule(rule RegisteredRule) {
	id := rule.Meta().ID
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("rule: rule with ID %q already registered", id))
	}
	registry[id] = rule
}

func GetRule(id string) (RegisteredRule, bool) {
	rule, exists := registry[id]
	return rule, exists
}

func AllRules() []RegisteredRule {
	rules := make([]RegisteredRule, 0, len(registry))
	for _, rule := range registry {
		rules = append(rules, rule)
	}
	return rules
}
