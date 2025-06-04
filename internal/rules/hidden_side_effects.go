package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type HiddenSideEffectsRule struct {
	Rule
}

func NewHiddenSideEffectsRule() *HiddenSideEffectsRule {
	return &HiddenSideEffectsRule{
		Rule: Rule{
			ID:          "hidden-side-effects",
			Name:        "Hidden Side Effects",
			Description: "Detects functions that perform side effects without making them explicit in their name, structure, or usage context. In functional programming, clarity around purity is essential for reasoning and testing.",
			Severity:    SeverityWarning,
		},
	}
}

func (r *HiddenSideEffectsRule) Meta() Rule {
	return r.Rule
}

func (r *HiddenSideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if !r.isFunction(node) {
		return nil
	}

	funcName := r.getFunctionName(node)
	if funcName == "" {
		return nil
	}

	sideEffects := r.analyzeSideEffects(node)

	if len(sideEffects) == 0 {
		return nil
	}

	if r.shouldBePureFunction(funcName) {
		return &Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf("Function '%s' appears to be pure based on its name but contains hidden side effects: %s. Consider making side effects explicit in the function name (e.g., 'save-user!', 'log-event!') or using 'doseq' for side-effect operations.",
				funcName, strings.Join(sideEffects, ", ")),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	if r.hasSideEffectIndicator(funcName) {
		return nil
	}

	if r.hasSignificantSideEffects(sideEffects) {
		return &Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf("Function '%s' performs side effects (%s) without explicit indication. Consider adding '!' suffix or using 'doseq' for side-effect operations to make the impurity explicit.",
				funcName, strings.Join(sideEffects, ", ")),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func (r *HiddenSideEffectsRule) isFunction(node *reader.RichNode) bool {
	return node.Type == reader.NodeList &&
		len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || node.Children[0].Value == "defn-")
}

func (r *HiddenSideEffectsRule) getFunctionName(node *reader.RichNode) string {
	if node.Type == reader.NodeList && len(node.Children) > 1 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || node.Children[0].Value == "defn-") {
		if node.Children[1].Type == reader.NodeSymbol {
			return node.Children[1].Value
		}
	}
	return ""
}

func (r *HiddenSideEffectsRule) analyzeSideEffects(node *reader.RichNode) []string {
	var sideEffects []string
	r.visitNodeForSideEffects(node, &sideEffects)
	return r.deduplicateEffects(sideEffects)
}

func (r *HiddenSideEffectsRule) visitNodeForSideEffects(node *reader.RichNode, effects *[]string) {
	if node == nil {
		return
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		funcCall := node.Children[0].Value

		if r.isIOOperation(funcCall) {
			*effects = append(*effects, "I/O operations")
		}

		if r.isStateMutation(funcCall) {
			*effects = append(*effects, "state mutations")
		}

		if r.isLogging(funcCall) {
			*effects = append(*effects, "logging")
		}

		if r.isDatabaseOperation(funcCall) {
			*effects = append(*effects, "database operations")
		}

		if r.isNetworkOperation(funcCall) {
			*effects = append(*effects, "network operations")
		}

		if r.isFileOperation(funcCall) {
			*effects = append(*effects, "file operations")
		}

		if r.isTimeDependentOperation(funcCall) {
			*effects = append(*effects, "time-dependent operations")
		}

		if r.isRandomOperation(funcCall) {
			*effects = append(*effects, "random operations")
		}
	}

	for _, child := range node.Children {
		r.visitNodeForSideEffects(child, effects)
	}
}

func (r *HiddenSideEffectsRule) isIOOperation(funcCall string) bool {
	ioOperations := map[string]bool{
		"println":     true,
		"print":       true,
		"printf":      true,
		"prn":         true,
		"pr":          true,
		"read":        true,
		"read-line":   true,
		"read-string": true,
		"spit":        true,
		"slurp":       true,
	}

	if strings.Contains(funcCall, "/") {
		parts := strings.Split(funcCall, "/")
		if len(parts) == 2 {
			namespace, function := parts[0], parts[1]
			if namespace == "io" || namespace == "clojure.java.io" {
				return true
			}
			return ioOperations[function]
		}
	}

	return ioOperations[funcCall]
}

func (r *HiddenSideEffectsRule) isStateMutation(funcCall string) bool {
	stateMutations := map[string]bool{
		"swap!":            true,
		"reset!":           true,
		"alter":            true,
		"commute":          true,
		"ref-set":          true,
		"send":             true,
		"send-off":         true,
		"deliver":          true,
		"compare-and-set!": true,
		"set!":             true,
	}

	if strings.Contains(funcCall, "/") {
		parts := strings.Split(funcCall, "/")
		if len(parts) == 2 {
			return stateMutations[parts[1]]
		}
	}

	return stateMutations[funcCall]
}

func (r *HiddenSideEffectsRule) isLogging(funcCall string) bool {
	loggingOperations := map[string]bool{
		"log":   true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
		"trace": true,
	}

	if strings.Contains(funcCall, "/") {
		parts := strings.Split(funcCall, "/")
		if len(parts) == 2 {
			namespace, function := parts[0], parts[1]

			if strings.Contains(namespace, "log") || namespace == "timbre" {
				return true
			}
			return loggingOperations[function]
		}
	}

	return loggingOperations[funcCall]
}

func (r *HiddenSideEffectsRule) isDatabaseOperation(funcCall string) bool {
	dbOperations := map[string]bool{
		"insert":  true,
		"update":  true,
		"delete":  true,
		"execute": true,
		"query":   true,
		"save":    true,
		"persist": true,
	}

	if strings.Contains(funcCall, "/") {
		parts := strings.Split(funcCall, "/")
		if len(parts) == 2 {
			namespace, function := parts[0], parts[1]

			if strings.Contains(namespace, "db") || strings.Contains(namespace, "sql") ||
				namespace == "korma" || namespace == "honeysql" || namespace == "next.jdbc" {
				return true
			}
			return dbOperations[function]
		}
	}

	return dbOperations[funcCall]
}

func (r *HiddenSideEffectsRule) isNetworkOperation(funcCall string) bool {
	networkOperations := map[string]bool{
		"get":     true,
		"post":    true,
		"put":     true,
		"delete":  true,
		"request": true,
		"send":    true,
		"publish": true,
	}

	if strings.Contains(funcCall, "/") {
		parts := strings.Split(funcCall, "/")
		if len(parts) == 2 {
			namespace, function := parts[0], parts[1]

			if strings.Contains(namespace, "http") || strings.Contains(namespace, "client") ||
				namespace == "clj-http" || namespace == "aleph" {
				return true
			}
			return networkOperations[function]
		}
	}

	return networkOperations[funcCall]
}

func (r *HiddenSideEffectsRule) isFileOperation(funcCall string) bool {
	fileOperations := map[string]bool{
		"delete-file": true,
		"copy":        true,
		"move":        true,
		"mkdir":       true,
		"rmdir":       true,
	}

	if strings.Contains(funcCall, "/") {
		parts := strings.Split(funcCall, "/")
		if len(parts) == 2 {
			namespace, function := parts[0], parts[1]
			if namespace == "io" || namespace == "clojure.java.io" || strings.Contains(namespace, "file") {
				return true
			}
			return fileOperations[function]
		}
	}

	return fileOperations[funcCall]
}

func (r *HiddenSideEffectsRule) isTimeDependentOperation(funcCall string) bool {
	timeOperations := map[string]bool{
		"now":                 true,
		"current-time":        true,
		"system-time":         true,
		"current-time-millis": true,
		"nano-time":           true,
	}

	if strings.Contains(funcCall, "/") {
		parts := strings.Split(funcCall, "/")
		if len(parts) == 2 {
			namespace, function := parts[0], parts[1]
			if strings.Contains(namespace, "time") || namespace == "clj-time" {
				return true
			}
			return timeOperations[function]
		}
	}

	return timeOperations[funcCall] || funcCall == "System/currentTimeMillis" || funcCall == "System/nanoTime"
}

func (r *HiddenSideEffectsRule) isRandomOperation(funcCall string) bool {
	randomOperations := map[string]bool{
		"rand":     true,
		"rand-int": true,
		"rand-nth": true,
		"random":   true,
		"shuffle":  true,
		"uuid":     true,
	}

	if strings.Contains(funcCall, "/") {
		parts := strings.Split(funcCall, "/")
		if len(parts) == 2 {
			namespace, function := parts[0], parts[1]
			if strings.Contains(namespace, "random") || strings.Contains(namespace, "uuid") {
				return true
			}
			return randomOperations[function]
		}
	}

	return randomOperations[funcCall] || strings.Contains(funcCall, "Random")
}

func (r *HiddenSideEffectsRule) shouldBePureFunction(funcName string) bool {

	if r.hasSideEffectIndicator(funcName) {
		return false
	}

	pureIndicators := []string{
		"calculate", "compute", "transform", "convert", "parse", "format",
		"validate", "check", "filter", "map", "reduce", "process",
		"get", "find", "search", "extract", "build", "create",
	}

	lowerName := strings.ToLower(funcName)
	for _, indicator := range pureIndicators {
		if strings.Contains(lowerName, indicator) {
			return true
		}
	}

	return false
}

func (r *HiddenSideEffectsRule) hasSideEffectIndicator(funcName string) bool {

	return strings.HasSuffix(funcName, "!") ||
		strings.Contains(strings.ToLower(funcName), "save") ||
		strings.Contains(strings.ToLower(funcName), "send") ||
		strings.Contains(strings.ToLower(funcName), "write") ||
		strings.Contains(strings.ToLower(funcName), "log") ||
		strings.Contains(strings.ToLower(funcName), "print") ||
		strings.Contains(strings.ToLower(funcName), "persist") ||
		strings.Contains(strings.ToLower(funcName), "update") ||
		strings.Contains(strings.ToLower(funcName), "insert") ||
		strings.Contains(strings.ToLower(funcName), "delete")
}

func (r *HiddenSideEffectsRule) hasSignificantSideEffects(sideEffects []string) bool {

	for _, effect := range sideEffects {
		if effect != "logging" {
			return true
		}
	}

	return len(sideEffects) > 2
}

func (r *HiddenSideEffectsRule) deduplicateEffects(effects []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, effect := range effects {
		if !seen[effect] {
			seen[effect] = true
			result = append(result, effect)
		}
	}

	return result
}

func init() {
	RegisterRule(NewHiddenSideEffectsRule())
}
