// Package rules implementa regras para detectar acessos a campos não existentes em mapas
// Esta regra específica identifica padrões de acesso que podem causar erros de runtime
package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// AccessingNonexistentMapFieldsRule detecta acessos potencialmente perigosos a campos de mapas
// Identifica padrões que podem causar NullPointerException ou comportamentos inesperados
type AccessingNonexistentMapFieldsRule struct {
	Rule
	CheckDirectKeywordAccess bool `json:"check_direct_keyword_access" yaml:"check_direct_keyword_access"` // Verifica acesso direto com keywords
	CheckThreadingMacros     bool `json:"check_threading_macros" yaml:"check_threading_macros"`           // Verifica threading macros perigosos
	CheckNestedAccess        bool `json:"check_nested_access" yaml:"check_nested_access"`                 // Verifica acessos aninhados
	MinNestingLevel          int  `json:"min_nesting_level" yaml:"min_nesting_level"`                     // Nível mínimo de aninhamento para alertar
}

func (r *AccessingNonexistentMapFieldsRule) Meta() Rule {
	return r.Rule
}

// Check analisa nós procurando por padrões de acesso perigoso a mapas
func (r *AccessingNonexistentMapFieldsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Verifica diferentes padrões de acesso perigoso
	if finding := r.checkDirectKeywordAccess(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkThreadingMacroAccess(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkGetInAccess(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkNestedMapAccess(node, filepath); finding != nil {
		return finding
	}

	return nil
}

// checkDirectKeywordAccess verifica acesso direto usando keywords sem verificação
func (r *AccessingNonexistentMapFieldsRule) checkDirectKeywordAccess(node *reader.RichNode, filepath string) *Finding {
	if !r.CheckDirectKeywordAccess {
		return nil
	}

	// Procura por padrões como (:keyword map) sem default
	if node.Type == reader.NodeList && len(node.Children) == 2 {
		firstChild := node.Children[0]
		secondChild := node.Children[1]

		// Verifica se é um acesso direto com keyword
		if firstChild.Type == reader.NodeKeyword && secondChild.Type == reader.NodeSymbol {
			keyword := firstChild.Value
			mapVar := secondChild.Value

			// Ignora casos com defaults ou verificações comuns
			if r.isCommonSafePattern(keyword, mapVar) {
				return nil
			}

			return &Finding{
				RuleID:   r.ID,
				Message:  r.formatDirectAccessMessage(keyword, mapVar),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

// checkThreadingMacroAccess verifica threading macros que podem falhar
func (r *AccessingNonexistentMapFieldsRule) checkThreadingMacroAccess(node *reader.RichNode, filepath string) *Finding {
	if !r.CheckThreadingMacros {
		return nil
	}

	// Procura por padrões como (-> map :field :nested-field)
	if node.Type == reader.NodeList && len(node.Children) >= 3 {
		firstChild := node.Children[0]

		if firstChild.Type == reader.NodeSymbol && (firstChild.Value == "->" || firstChild.Value == "->>") {
			// Conta quantos acessos de keyword consecutivos existem
			keywordCount := 0
			var keywords []string

			for i := 2; i < len(node.Children); i++ {
				child := node.Children[i]
				if child.Type == reader.NodeKeyword {
					keywordCount++
					keywords = append(keywords, child.Value)
				} else {
					break
				}
			}

			// Se há múltiplos acessos de keyword, pode ser perigoso
			if keywordCount >= r.MinNestingLevel {
				return &Finding{
					RuleID:   r.ID,
					Message:  r.formatThreadingMessage(firstChild.Value, keywords),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}

	return nil
}

// checkGetInAccess verifica uso de get-in sem defaults
func (r *AccessingNonexistentMapFieldsRule) checkGetInAccess(node *reader.RichNode, filepath string) *Finding {
	// Procura por padrões como (get-in map [:path]) sem default
	if node.Type == reader.NodeList && len(node.Children) == 3 {
		firstChild := node.Children[0]

		if firstChild.Type == reader.NodeSymbol && firstChild.Value == "get-in" {
			// get-in sem default value é potencialmente perigoso
			return &Finding{
				RuleID:   r.ID,
				Message:  "get-in without default value detected. Consider providing a default to handle missing keys safely.",
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

// checkNestedMapAccess verifica acessos aninhados perigosos
func (r *AccessingNonexistentMapFieldsRule) checkNestedMapAccess(node *reader.RichNode, filepath string) *Finding {
	if !r.CheckNestedAccess {
		return nil
	}

	// Procura por padrões como ((:nested-key (:key map)))
	if r.isNestedKeywordAccess(node, 0) >= r.MinNestingLevel {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Deeply nested map access detected without safety checks. Consider using get-in with defaults or validating intermediate values.",
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

// isNestedKeywordAccess conta o nível de aninhamento de acessos com keywords
func (r *AccessingNonexistentMapFieldsRule) isNestedKeywordAccess(node *reader.RichNode, depth int) int {
	if node.Type == reader.NodeList && len(node.Children) == 2 {
		firstChild := node.Children[0]
		secondChild := node.Children[1]

		if firstChild.Type == reader.NodeKeyword {
			// Se o segundo filho também é um acesso aninhado, conta recursivamente
			if secondChild.Type == reader.NodeList {
				return 1 + r.isNestedKeywordAccess(secondChild, depth+1)
			}
			// Se é um símbolo (variável), conta como 1 nível
			if secondChild.Type == reader.NodeSymbol {
				return 1
			}
		}
	}

	return 0
}

// isCommonSafePattern verifica se é um padrão comumente seguro
func (r *AccessingNonexistentMapFieldsRule) isCommonSafePattern(keyword, mapVar string) bool {
	// Ignora padrões comuns que geralmente são seguros
	safeKeywords := []string{":id", ":type", ":status", ":name"}
	for _, safe := range safeKeywords {
		if keyword == safe {
			return true
		}
	}

	// Ignora variáveis que sugerem validação prévia
	safeVarPatterns := []string{"validated-", "checked-", "safe-", "verified-"}
	for _, pattern := range safeVarPatterns {
		if strings.HasPrefix(mapVar, pattern) {
			return true
		}
	}

	return false
}

// formatDirectAccessMessage formata mensagem para acesso direto
func (r *AccessingNonexistentMapFieldsRule) formatDirectAccessMessage(keyword, mapVar string) string {
	return fmt.Sprintf(
		"Direct map access (%s %s) without safety check detected. Consider using (get %s %s default-value) or validating the map structure first.",
		keyword, mapVar, mapVar, keyword,
	)
}

// formatThreadingMessage formata mensagem para threading macros
func (r *AccessingNonexistentMapFieldsRule) formatThreadingMessage(macro string, keywords []string) string {
	keywordStr := strings.Join(keywords, " ")
	return fmt.Sprintf(
		"Potentially unsafe threading macro (%s) with multiple keyword accesses [%s]. Consider using get-in with defaults or validating intermediate values.",
		macro, keywordStr,
	)
}

// init registra a regra de accessing nonexistent map fields
func init() {
	defaultRule := &AccessingNonexistentMapFieldsRule{
		Rule: Rule{
			ID:          "accessing-nonexistent-map-fields",
			Name:        "Accessing Non-Existent Map Fields",
			Description: "Detects potentially unsafe access to map fields that may not exist. Suggests using safe access patterns with defaults or validation to prevent runtime errors.",
			Severity:    SeverityWarning,
		},
		CheckDirectKeywordAccess: true,
		CheckThreadingMacros:     true,
		CheckNestedAccess:        true,
		MinNestingLevel:          2, // Alerta para 2+ níveis de aninhamento
	}

	RegisterRule(defaultRule)
}
