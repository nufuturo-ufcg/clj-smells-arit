package rules

import (
	"fmt"
	"sort"
	"sync"

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

var (
	registry   = make(map[string]RegisteredRule)
	registryMu sync.RWMutex
)

func RegisterRule(rule RegisteredRule) {
	registryMu.Lock()
	defer registryMu.Unlock()

	id := rule.Meta().ID
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("rule: rule with ID %q already registered", id))
	}
	registry[id] = rule
}

func GetRule(id string) (RegisteredRule, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	rule, exists := registry[id]
	return rule, exists
}

func AllRules() []RegisteredRule {
	registryMu.RLock()
	defer registryMu.RUnlock()

	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	rules := make([]RegisteredRule, 0, len(registry))
	for _, id := range ids {
		rules = append(rules, registry[id])
	}

	return rules
}
