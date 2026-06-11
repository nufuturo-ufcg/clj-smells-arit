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
	Group       string   `json:"group" yaml:"group"`
}

func (r *Rule) IsInside(context map[string]interface{}, formNames ...string) bool {
	return isInsideContext(context, formNames)
}

func isInsideContext(context map[string]interface{}, formNames []string) bool {
	enclosingForms, ok := context["enclosingForms"].([]string)
	if !ok {
		return false
	}
	for _, enclosing := range enclosingForms {
		for _, target := range formNames {
			if enclosing == target {
				return true
			}
		}
	}
	return false
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

var RuleGroups = map[string]string{
	"blocking-inside-go":                  "clojure-specific",
	"direct-use-of-clojure-lang-rt":       "clojure-specific",
	"implicit-namespace-dependencies":     "clojure-specific",
	"improper-emptiness-check":            "clojure-specific",
	"misuse-of-channel-closing-semantics": "clojure-specific",
	"monolithic-namespace-split":          "clojure-specific",
	"multiple-evaluation-in-macros":       "clojure-specific",
	"namespace-load-side-effects":         "clojure-specific",
	"private-multimethods":                "clojure-specific",
	"production-doall":                    "clojure-specific",
	"redundant-do-block":                  "clojure-specific",
	"single-segment-namespace":            "clojure-specific",
	"thread-ignorance":                    "clojure-specific",
	"unnecessary-into":                    "clojure-specific",
	"verbose-checks":                      "clojure-specific",
	"namespaced-keys-neglect":             "clojure-specific",
	"library-locker":                      "clojure-specific",
	"accessing-nonexistent-map-fields":    "clojure-specific",
	"conditional-build-up":                "clojure-specific",
	"nested-forms":                        "clojure-specific",

	"immutability-violation":            "functional",
	"lazy-side-effects":                 "functional",
	"explicit-recursion":                "functional",
	"inefficient-filtering":             "functional",
	"potentially-inefficient-generator": "functional",
	"premature-optimization":            "functional",
	"trivial-lambda":                    "functional",
	"underutilizing-features":           "functional",
	"overuse-of-high-order-functions":   "functional",
	"overabstracted-composition":        "functional",

	"circular-dependency":          "traditional",
	"comments":                     "traditional",
	"data-clumps":                  "traditional",
	"deeply-nested":                "traditional",
	"direct-external-schema-usage": "traditional",
	"divergent-change":             "traditional",
	"duplicated-code":              "traditional",
	"external-data-coupling":       "traditional",
	"feature-envy":                 "traditional",
	"hidden-side-effects":          "traditional",
	"inappropriate-collection":     "traditional",
	"linear-collection-scan":       "traditional",
	"long-function":                "traditional",
	"long-parameter-list":          "traditional",
	"message-chains":               "traditional",
	"middle-man":                   "traditional",
	"positional-return-values":     "traditional",
	"primitive-obsession":          "traditional",
	"shotgun-surgery":              "traditional",
	"string-map-keys":              "traditional",
	"unnecessary-abstraction":      "traditional",
}

func GetRuleGroup(id string) string {
	if grp, ok := RuleGroups[id]; ok {
		return grp
	}
	return "clojure-specific" // default group
}
