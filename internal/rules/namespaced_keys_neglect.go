package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// NamespacedKeysNeglectRule detecta negligência no uso de namespaced keys
// Identifica keywords que deveriam usar namespaces para evitar colisões e melhorar clareza
type NamespacedKeysNeglectRule struct {
	Rule
}

// KeywordContext representa o contexto onde uma keyword é usada
type KeywordContext struct {
	Type        string // "map-key", "spec-key", "destructuring", "function-call"
	Scope       string // "local", "api", "database", "global"
	Suggestion  string
	Confidence  string // "high", "medium", "low"
	Description string
}

// Padrões de keywords que comumente precisam de namespace
var commonGlobalKeywords = map[string]bool{
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

// Padrões que indicam uso em APIs ou persistência
var apiPatterns = []string{
	"defapi", "defroute", "POST", "GET", "PUT", "DELETE", "PATCH",
	"defentity", "defschema", "defspec", "s/def",
	"insert", "update", "select", "delete", "query",
	"create-table", "alter-table", "drop-table",
}

func (r *NamespacedKeysNeglectRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
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

	// Determina severidade baseada no contexto e confiança
	severity := r.determineSeverity(keywordContext)

	message := fmt.Sprintf("Non-namespaced keyword '%s' detected in %s context. %s. Suggestion: %s",
		keywordValue, keywordContext.Type, keywordContext.Description, keywordContext.Suggestion)

	return &Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: severity,
	}
}

// isKeyword verifica se o nó é uma keyword
func (r *NamespacedKeysNeglectRule) isKeyword(node *reader.RichNode) bool {
	return node != nil && node.Type == reader.NodeKeyword
}

// isAlreadyNamespaced verifica se a keyword já tem namespace
func (r *NamespacedKeysNeglectRule) isAlreadyNamespaced(keyword string) bool {
	// Remove o : inicial
	if strings.HasPrefix(keyword, ":") {
		keyword = keyword[1:]
	}

	// Verifica se contém namespace (formato namespace/name ou namespace.subns/name)
	return strings.Contains(keyword, "/") || 
		   (strings.Contains(keyword, ".") && strings.Contains(keyword, "/"))
}

// analyzeKeywordContext analisa o contexto onde a keyword é usada
func (r *NamespacedKeysNeglectRule) analyzeKeywordContext(node *reader.RichNode, context map[string]interface{}) *KeywordContext {
	keywordValue := strings.TrimPrefix(node.Value, ":")
	
	// Verifica se é uma keyword comum que deveria ter namespace
	if r.isCommonGlobalKeyword(keywordValue) {
		return &KeywordContext{
			Type:        "global-data",
			Scope:       "global",
			Suggestion:  fmt.Sprintf("Consider using namespaced keyword like :myapp.user/%s or :myapp/%s", keywordValue, keywordValue),
			Confidence:  "high",
			Description: "This is a common keyword that often appears across system boundaries and should be namespaced to avoid collisions",
		}
	}

	// Analisa o contexto do nó pai
	parentContext := r.getParentContext(node, context)
	
	if parentContext != nil {
		return parentContext
	}

	// Verifica se está em contexto de spec
	if r.isInSpecContext(node, context) {
		return &KeywordContext{
			Type:        "spec-key",
			Scope:       "api",
			Suggestion:  fmt.Sprintf("Use namespaced keyword like :myapp.spec/%s for spec definitions", keywordValue),
			Confidence:  "high",
			Description: "Spec keywords should be namespaced to avoid conflicts and improve discoverability",
		}
	}

	// Verifica se está em contexto de API
	if r.isInAPIContext(node, context) {
		return &KeywordContext{
			Type:        "api-key",
			Scope:       "api",
			Suggestion:  fmt.Sprintf("Use namespaced keyword like :myapp.api/%s for API data", keywordValue),
			Confidence:  "medium",
			Description: "API keywords should be namespaced for better traceability across system boundaries",
		}
	}

	// Verifica se está em contexto de mapa com múltiplas chaves
	if r.isInLargeMapContext(node, context) {
		return &KeywordContext{
			Type:        "map-key",
			Scope:       "local",
			Suggestion:  fmt.Sprintf("Consider namespacing as :myapp.domain/%s if this data crosses boundaries", keywordValue),
			Confidence:  "low",
			Description: "Large maps with many keys benefit from namespacing for clarity and collision avoidance",
		}
	}

	return nil
}

// isCommonGlobalKeyword verifica se é uma keyword comum que deveria ter namespace
func (r *NamespacedKeysNeglectRule) isCommonGlobalKeyword(keyword string) bool {
	return commonGlobalKeywords[keyword]
}

// getParentContext analisa o contexto do nó pai
func (r *NamespacedKeysNeglectRule) getParentContext(node *reader.RichNode, context map[string]interface{}) *KeywordContext {
	// Esta função seria implementada para analisar o contexto do AST
	// Por simplicidade, vamos focar nos padrões mais óbvios
	return nil
}

// isInSpecContext verifica se está em contexto de clojure.spec
func (r *NamespacedKeysNeglectRule) isInSpecContext(node *reader.RichNode, context map[string]interface{}) bool {
	// Verifica se há indicadores de spec no contexto
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

// isInAPIContext verifica se está em contexto de API
func (r *NamespacedKeysNeglectRule) isInAPIContext(node *reader.RichNode, context map[string]interface{}) bool {
	if contextStr, ok := context["function_name"].(string); ok {
		for _, pattern := range apiPatterns {
			if strings.Contains(contextStr, pattern) {
				return true
			}
		}
	}
	
	// Verifica se está em namespace que sugere API
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

// isInLargeMapContext verifica se está em um mapa grande
func (r *NamespacedKeysNeglectRule) isInLargeMapContext(node *reader.RichNode, context map[string]interface{}) bool {
	// Verifica se o mapa pai tem muitas chaves (indicativo de estrutura complexa)
	if mapSize, ok := context["map_size"].(int); ok {
		return mapSize >= 5 // Mapas com 5+ chaves podem se beneficiar de namespacing
	}
	return false
}

// determineSeverity determina a severidade baseada no contexto
func (r *NamespacedKeysNeglectRule) determineSeverity(ctx *KeywordContext) Severity {
	switch ctx.Confidence {
	case "high":
		if ctx.Scope == "global" || ctx.Scope == "api" {
			return SeverityWarning
		}
		return SeverityInfo
	case "medium":
		return SeverityInfo
	default:
		return SeverityHint
	}
}

// Funções auxiliares para análise de padrões

// isInDatabaseContext verifica se está em contexto de banco de dados
func (r *NamespacedKeysNeglectRule) isInDatabaseContext(node *reader.RichNode, context map[string]interface{}) bool {
	if contextStr, ok := context["function_name"].(string); ok {
		dbPatterns := []string{
			"insert", "update", "select", "delete", "query",
			"create-table", "alter-table", "defentity",
			"jdbc", "sql", "db", "database",
		}
		for _, pattern := range dbPatterns {
			if strings.Contains(strings.ToLower(contextStr), pattern) {
				return true
			}
		}
	}
	return false
}

// isInConfigContext verifica se está em contexto de configuração
func (r *NamespacedKeysNeglectRule) isInConfigContext(node *reader.RichNode, context map[string]interface{}) bool {
	if contextStr, ok := context["function_name"].(string); ok {
		configPatterns := []string{"config", "settings", "env", "properties"}
		for _, pattern := range configPatterns {
			if strings.Contains(strings.ToLower(contextStr), pattern) {
				return true
			}
		}
	}
	return false
}

// hasSnakeCasePattern verifica se segue o padrão snake_case (mais portável)
func (r *NamespacedKeysNeglectRule) hasSnakeCasePattern(keyword string) bool {
	// Remove o : inicial
	keyword = strings.TrimPrefix(keyword, ":")
	
	// Verifica se usa snake_case
	snakeCaseRegex := regexp.MustCompile(`^[a-z][a-z0-9_]*[a-z0-9]$`)
	return snakeCaseRegex.MatchString(keyword)
}

// hasLispCasePattern verifica se segue o padrão lisp-case (idiomático Clojure)
func (r *NamespacedKeysNeglectRule) hasLispCasePattern(keyword string) bool {
	// Remove o : inicial
	keyword = strings.TrimPrefix(keyword, ":")
	
	// Verifica se usa lisp-case
	lispCaseRegex := regexp.MustCompile(`^[a-z][a-z0-9\-]*[a-z0-9]$`)
	return lispCaseRegex.MatchString(keyword)
}

// suggestNamespacing sugere um namespace apropriado baseado no contexto
func (r *NamespacedKeysNeglectRule) suggestNamespacing(keyword string, context *KeywordContext) string {
	keyword = strings.TrimPrefix(keyword, ":")
	
	switch context.Scope {
	case "global":
		return fmt.Sprintf(":myapp.core/%s", keyword)
	case "api":
		return fmt.Sprintf(":myapp.api/%s", keyword)
	case "database":
		return fmt.Sprintf(":myapp.db/%s", keyword)
	case "config":
		return fmt.Sprintf(":myapp.config/%s", keyword)
	default:
		return fmt.Sprintf(":myapp.domain/%s", keyword)
	}
}

func (r *NamespacedKeysNeglectRule) Meta() Rule {
	return r.Rule
}

func init() {
	rule := &NamespacedKeysNeglectRule{
		Rule: Rule{
			ID:          "namespaced-keys-neglect",
			Name:        "Namespaced Keys Neglect",
			Description: "Detects keywords that should use namespaces to avoid collisions and improve code clarity. Namespaced keywords provide better traceability across system boundaries and reduce ambiguity, especially for common attribute names like :id, :name, :email, etc.",
			Severity:    SeverityInfo,
		},
	}

	RegisterRule(rule)
} 