// Package rules implementa regras para promover estilo idiomático em Clojure
// Esta regra específica detecta uso de strings como chaves de mapa quando keywords seriam mais apropriadas
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// StringMapKeysRule detecta uso de strings como chaves em mapas literais
// Promove o uso de keywords que são mais eficientes e idiomáticas em Clojure
type StringMapKeysRule struct {
	Rule
}

func (r *StringMapKeysRule) Meta() Rule {
	return r.Rule
}

// init registra a regra de chaves de mapa com configurações padrão
// Configurada como INFO pois é uma questão de estilo e performance
func init() {
	RegisterRule(&StringMapKeysRule{
		Rule: Rule{
			ID:          "string-map-keys",
			Name:        "String Keys in Map Literal",
			Description: "Map literals should use keywords (:key) instead of strings (\"key\") as keys for better performance and idiomatic style.",

			Severity: SeverityInfo,
		},
	})
}

// Check analisa mapas literais procurando por chaves que são strings
// Keywords são preferíveis por serem mais eficientes e idiomáticas
func (r *StringMapKeysRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Verifica apenas nós de mapa
	if node.Type != reader.NodeMap {
		return nil
	}

	// Mapas devem ter número par de filhos (chave-valor, chave-valor...)
	if len(node.Children)%2 != 0 {
		// Mapa malformado - não é responsabilidade desta regra
		return nil
	}

	// Examina cada par chave-valor no mapa
	for i := 0; i < len(node.Children); i += 2 {
		keyNode := node.Children[i]

		// Detecta chaves que são strings
		if keyNode.InferredType == "String" {
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Map literal uses string key %q instead of a keyword. Consider using ':%s' for idiomatic Clojure.", keyNode.Value, keyNode.Value),
				Filepath: filepath,
				Location: keyNode.Location,
				Severity: r.Severity,
			}

		}
	}

	return nil
}
