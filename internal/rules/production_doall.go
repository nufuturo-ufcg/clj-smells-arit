// Package rules implementa regras para detectar uso de doall em código de produção
// Esta regra específica identifica chamadas de doall que podem causar problemas de performance
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// ProductionDoallRule detecta uso de doall em código de produção
// doall força a realização de sequências lazy, podendo causar problemas de memória e performance
type ProductionDoallRule struct {
	Rule
	AllowInTests   bool `json:"allow_in_tests" yaml:"allow_in_tests"`       // Permite doall em testes
	AllowInDevCode bool `json:"allow_in_dev_code" yaml:"allow_in_dev_code"` // Permite doall em código de desenvolvimento
	AllowInREPL    bool `json:"allow_in_repl" yaml:"allow_in_repl"`         // Permite doall em contexto REPL
}

func (r *ProductionDoallRule) Meta() Rule {
	return r.Rule
}

// isTestFile verifica se o arquivo parece ser um arquivo de teste
// baseado no nome do arquivo ou namespace
func (r *ProductionDoallRule) isTestFile(filepath string) bool {
	// Verifica se o arquivo contém indicadores de teste no nome
	testIndicators := []string{
		"test",
		"spec",
		"check",
		"_test",
		"-test",
		"test_",
		"test-",
	}

	for _, indicator := range testIndicators {
		if contains(filepath, indicator) {
			return true
		}
	}

	return false
}

// isDevCode verifica se o arquivo parece ser código de desenvolvimento
// baseado no caminho ou namespace
func (r *ProductionDoallRule) isDevCode(filepath string) bool {
	devIndicators := []string{
		"dev",
		"development",
		"debug",
		"repl",
		"scratch",
		"playground",
		"example",
		"demo",
	}

	for _, indicator := range devIndicators {
		if contains(filepath, indicator) {
			return true
		}
	}

	return false
}

// contains verifica se uma string contém um substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				anySubstring(s, substr)))
}

// anySubstring verifica se substr está presente em s
func anySubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// isInREPLContext verifica se estamos em contexto REPL
// baseado em indicadores no código ou namespace
func (r *ProductionDoallRule) isInREPLContext(node *reader.RichNode, filepath string) bool {
	// Verifica se o arquivo tem indicadores de REPL
	replIndicators := []string{
		"repl",
		"user", // namespace padrão do REPL
		"scratch",
	}

	for _, indicator := range replIndicators {
		if contains(filepath, indicator) {
			return true
		}
	}

	// Verifica se estamos dentro de um namespace típico de REPL
	// (isso seria mais preciso com análise do namespace, mas simplificamos aqui)
	return false
}

// Check analisa nós procurando por chamadas de doall
func (r *ProductionDoallRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Verifica se é uma chamada de função
	if node.Type != reader.NodeList || len(node.Children) < 1 {
		return nil
	}

	// Verifica se o primeiro elemento é o símbolo 'doall'
	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || firstChild.Value != "doall" {
		return nil
	}

	// Verifica contextos onde doall é permitido
	if r.AllowInTests && r.isTestFile(filepath) {
		return nil
	}

	if r.AllowInDevCode && r.isDevCode(filepath) {
		return nil
	}

	if r.AllowInREPL && r.isInREPLContext(node, filepath) {
		return nil
	}

	// Determina o contexto específico onde doall foi encontrado
	contextDescription := r.getContextDescription(node, context)

	// Cria mensagem detalhada sobre o problema
	message := fmt.Sprintf(
		"Usage of 'doall' detected in production code%s. "+
			"'doall' forces realization of lazy sequences which can cause memory issues and performance problems. "+
			"Consider using eager operations (mapv, into, vec) or restructuring to avoid forcing evaluation. "+
			"If this is intentional for debugging/testing, consider moving to test files or dev-specific code.",
		contextDescription,
	)

	return &Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

// getContextDescription analisa o contexto onde doall aparece para dar feedback mais específico
func (r *ProductionDoallRule) getContextDescription(node *reader.RichNode, context map[string]interface{}) string {
	// Verifica se doall está dentro de uma função específica
	if parent, ok := context["parent"]; ok {
		if parentNode, ok := parent.(*reader.RichNode); ok {
			if parentNode.Type == reader.NodeList && len(parentNode.Children) > 0 {
				if parentFirstChild := parentNode.Children[0]; parentFirstChild.Type == reader.NodeSymbol {
					switch parentFirstChild.Value {
					case "defn", "defn-":
						if len(parentNode.Children) > 1 && parentNode.Children[1].Type == reader.NodeSymbol {
							return fmt.Sprintf(" in function '%s'", parentNode.Children[1].Value)
						}
						return " in function definition"
					case "let", "when", "if":
						return fmt.Sprintf(" in %s form", parentFirstChild.Value)
					case "map", "mapcat", "filter":
						return fmt.Sprintf(" in %s operation (nested lazy operation)", parentFirstChild.Value)
					}
				}
			}
		}
	}

	// Analisa argumentos de doall para contexto adicional
	if len(node.Children) > 1 {
		argNode := node.Children[1]
		if argNode.Type == reader.NodeList && len(argNode.Children) > 0 {
			if firstArg := argNode.Children[0]; firstArg.Type == reader.NodeSymbol {
				switch firstArg.Value {
				case "map", "filter", "remove", "mapcat":
					return fmt.Sprintf(" on %s result", firstArg.Value)
				case "for":
					return " on list comprehension result"
				}
			}
		}
	}

	return ""
}

// init registra a regra de production doall com configurações padrão
func init() {
	defaultRule := &ProductionDoallRule{
		Rule: Rule{
			ID:          "production-doall",
			Name:        "Production doall Usage",
			Description: "Detects usage of 'doall' in production code. 'doall' forces realization of lazy sequences which can cause memory issues and performance problems. Consider using eager operations or restructuring code to avoid forcing evaluation.",
			Severity:    SeverityWarning,
		},
		AllowInTests:   false, // Por padrão, NÃO permite em testes - detecta também em arquivos de teste
		AllowInDevCode: false, // Por padrão, NÃO permite em código de desenvolvimento - mais restritivo
		AllowInREPL:    true,  // Por padrão, permite em contexto REPL
	}

	RegisterRule(defaultRule)
}
