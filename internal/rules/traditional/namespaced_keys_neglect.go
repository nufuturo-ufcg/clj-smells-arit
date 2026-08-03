package traditional

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"
	"strings"
	"sync"

	"github.com/thlaurentino/arit/internal/reader"
)

var (
	commonGlobalKeywords map[string]bool
	apiPatterns          []string
	namespacedKeysOnce   sync.Once
)

func initNamespacedKeysMaps() {
	namespacedKeysOnce.Do(func() {
		commonGlobalKeywords = map[string]bool{
			"id":         true,
			"name":       true,
			"email":      true,
			"password":   true,
			"username":   true,
			"first-name": true,
			"last-name":  true,
			"created-at": true,
			"updated-at": true,
			"status":     true,
			"type":       true,
			"value":      true,
			"data":       true,
			"config":     true,
			"settings":   true,
			"user":       true,
			"admin":      true,
			"role":       true,
			"permission": true,
			"token":      true,
			"session":    true,
			"error":      true,
			"message":    true,
			"code":       true,
			"result":     true,
			"response":   true,
			"request":    true,
		}

		apiPatterns = []string{
			"defapi", "defroute", "POST", "GET", "PUT", "DELETE", "PATCH",
			"defentity", "defschema", "defspec", "s/def",
			"insert", "update", "select", "delete", "query",
			"create-table", "alter-table", "drop-table",
		}
	})
}

type NamespacedKeysNeglectRule struct {
	rules.Rule
}

type KeywordContext struct {
	Type        string
	Scope       string
	Suggestion  string
	Confidence  string
	Description string
}

func (r *NamespacedKeysNeglectRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	initNamespacedKeysMaps()

	// Ignore test files, dev environments, and test support scripts
	if strings.HasSuffix(filepath, "_test.clj") || strings.Contains(filepath, "/dev/") || strings.Contains(filepath, "/test/") || strings.Contains(filepath, "/int-test/") || strings.Contains(filepath, "/support/") {
		return nil
	}

	if !r.isKeyword(node) {
		return nil
	}

	keywordValue := node.Value
	if r.isAlreadyNamespaced(keywordValue) {
		return nil
	}

	keywordContext := r.analyzeKeywordContext(node, context)
	if keywordContext == nil {
		return nil
	}

	severity := r.determineSeverity(keywordContext)

	message := fmt.Sprintf("Non-namespaced keyword '%s' detected in %s context. %s. Suggestion: %s",
		keywordValue, keywordContext.Type, keywordContext.Description, keywordContext.Suggestion)

	return &rules.Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: severity,
	}
}

func (r *NamespacedKeysNeglectRule) isKeyword(node *reader.RichNode) bool {
	return node != nil && node.Type == reader.NodeKeyword
}

func (r *NamespacedKeysNeglectRule) isAlreadyNamespaced(keyword string) bool {

	keyword = strings.TrimPrefix(keyword, ":")

	return strings.Contains(keyword, "/") ||
		(strings.Contains(keyword, ".") && strings.Contains(keyword, "/"))
}

func (r *NamespacedKeysNeglectRule) analyzeKeywordContext(node *reader.RichNode, context map[string]interface{}) *KeywordContext {
	keywordValue := strings.TrimPrefix(node.Value, ":")

	if r.isInSpecContext(node, context) {
		return &KeywordContext{
			Type:        "spec-key",
			Scope:       "api",
			Suggestion:  fmt.Sprintf("Use namespaced keyword like :myapp.spec/%s for spec definitions", keywordValue),
			Confidence:  "high",
			Description: "Spec keywords should be namespaced to avoid conflicts and improve discoverability",
		}
	}

	if r.isInAPIContext(node, context) {
		return &KeywordContext{
			Type:        "api-key",
			Scope:       "api",
			Suggestion:  fmt.Sprintf("Use namespaced keyword like :myapp.api/%s for API data", keywordValue),
			Confidence:  "medium",
			Description: "API keywords should be namespaced for better traceability across system boundaries",
		}
	}

	return nil
}

func (r *NamespacedKeysNeglectRule) isCommonGlobalKeyword(keyword string) bool {
	return commonGlobalKeywords[keyword]
}

func (r *NamespacedKeysNeglectRule) getParentContext(node *reader.RichNode, context map[string]interface{}) *KeywordContext {

	return nil
}

func (r *NamespacedKeysNeglectRule) isInSpecContext(node *reader.RichNode, context map[string]interface{}) bool {

	if contextStr, ok := context["function_name"].(string); ok {
		specIndicators := []string{"s/def", "defspec", "spec/def", "s/keys", "s/valid?"}
		for _, indicator := range specIndicators {
			if strings.Contains(contextStr, indicator) {
				return true
			}
		}
	}
	return false
}

func (r *NamespacedKeysNeglectRule) isInAPIContext(node *reader.RichNode, context map[string]interface{}) bool {
	if contextStr, ok := context["function_name"].(string); ok {
		for _, pattern := range apiPatterns {
			if strings.Contains(contextStr, pattern) {
				return true
			}
		}
	}

	if ns, ok := context["namespace"].(string); ok {
		apiNamespaces := []string{"api", "routes", "handlers", "endpoints", "rest", "graphql"}
		for _, apiNs := range apiNamespaces {
			if strings.Contains(strings.ToLower(ns), apiNs) {
				return true
			}
		}
	}

	return false
}

func (r *NamespacedKeysNeglectRule) isInLargeMapContext(node *reader.RichNode, context map[string]interface{}) bool {

	if mapSize, ok := context["map_size"].(int); ok {
		return mapSize >= 5
	}
	return false
}

func (r *NamespacedKeysNeglectRule) determineSeverity(ctx *KeywordContext) rules.Severity {
	switch ctx.Confidence {
	case "high":
		if ctx.Scope == "global" || ctx.Scope == "api" {
			return rules.SeverityWarning
		}
		return rules.SeverityInfo
	case "medium":
		return rules.SeverityInfo
	default:
		return rules.SeverityHint
	}
}

func (r *NamespacedKeysNeglectRule) Meta() rules.Rule {
	return r.Rule
}

func init() {
	rule := &NamespacedKeysNeglectRule{
		Rule: rules.Rule{
			ID:          "namespaced-keys-neglect",
			Name:        "Namespaced Keys Neglect",
			Description: "Detects keywords that should use namespaces to avoid collisions and improve code clarity. Namespaced keywords provide better traceability across system boundaries and reduce ambiguity, especially for common attribute names like :id, :name, :email, etc.",
			Severity:    rules.SeverityInfo,
		},
	}

	rules.RegisterRule(rule)
}