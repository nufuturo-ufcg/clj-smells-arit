package rules

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

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

type registrySnapshot struct {
	rules  map[string]RegisteredRule
	sorted []RegisteredRule
	allIDs []string
}

var (
	registry       = make(map[string]RegisteredRule)
	registryMu     sync.Mutex
	cachedSnapshot atomic.Value
)

func RegisterRule(rule RegisteredRule) {
	registryMu.Lock()
	defer registryMu.Unlock()

	id := rule.Meta().ID
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("rule: rule with ID %q already registered", id))
	}
	registry[id] = rule

	cachedSnapshot.Store((*registrySnapshot)(nil))
}

func getOrCreateSnapshot() *registrySnapshot {

	if val := cachedSnapshot.Load(); val != nil {
		if snapshot, ok := val.(*registrySnapshot); ok && snapshot != nil {
			return snapshot
		}
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if val := cachedSnapshot.Load(); val != nil {
		if snapshot, ok := val.(*registrySnapshot); ok && snapshot != nil {
			return snapshot
		}
	}

	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	rules := make([]RegisteredRule, 0, len(registry))
	rulesCopy := make(map[string]RegisteredRule, len(registry))

	for _, id := range ids {
		rule := registry[id]
		rules = append(rules, rule)
		rulesCopy[id] = rule
	}

	snapshot := &registrySnapshot{
		rules:  rulesCopy,
		sorted: rules,
		allIDs: ids,
	}

	cachedSnapshot.Store(snapshot)
	return snapshot
}

func GetRule(id string) (RegisteredRule, bool) {
	snapshot := getOrCreateSnapshot()
	if snapshot == nil || snapshot.rules == nil {

		registryMu.Lock()
		defer registryMu.Unlock()
		rule, exists := registry[id]
		return rule, exists
	}

	rule, exists := snapshot.rules[id]
	return rule, exists
}

func AllRules() []RegisteredRule {
	snapshot := getOrCreateSnapshot()
	if snapshot == nil || snapshot.sorted == nil {

		registryMu.Lock()
		defer registryMu.Unlock()

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

	result := make([]RegisteredRule, len(snapshot.sorted))
	copy(result, snapshot.sorted)
	return result
}
