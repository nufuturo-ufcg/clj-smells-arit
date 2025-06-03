// Package rules implementa regra para detectar formas aninhadas excessivamente
// Esta regra identifica aninhamento excessivo de formas como let, when, if que tornam o código difícil de ler
package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// NestedFormsRule detecta aninhamento excessivo de formas
// Formas aninhadas demais tornam o código difícil de ler e manter
type NestedFormsRule struct {
	Rule
	MaxDepth       int      `json:"max_depth" yaml:"max_depth"`                 // Profundidade máxima permitida
	TrackedForms   []string `json:"tracked_forms" yaml:"tracked_forms"`         // Formas a serem rastreadas
	MinFormsInPath int      `json:"min_forms_in_path" yaml:"min_forms_in_path"` // Mínimo de formas no caminho para reportar
}

// NestingPath representa um caminho de aninhamento
type NestingPath struct {
	Forms []string
	Depth int
	Nodes []*reader.RichNode
}

func (r *NestedFormsRule) Meta() Rule {
	return r.Rule
}

// Check analisa nós procurando por aninhamento excessivo
func (r *NestedFormsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Só processa formas rastreadas
	if r.isTrackedForm(node) {
		// Constrói o caminho de aninhamento atual
		path := r.buildNestingPath(node, context)

		// Verifica se excede os limites configurados
		if r.exceedsLimits(path) {
			suggestion := r.getSuggestionForPath(path)

			message := fmt.Sprintf(
				"Excessive nesting detected (depth: %d, forms: %s). %s",
				path.Depth,
				strings.Join(path.Forms, " → "),
				suggestion,
			)

			return &Finding{
				RuleID:   r.ID,
				Message:  message,
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

// buildNestingPath constrói o caminho de aninhamento a partir do contexto
func (r *NestedFormsRule) buildNestingPath(node *reader.RichNode, context map[string]interface{}) *NestingPath {
	path := &NestingPath{
		Forms: []string{},
		Depth: 0,
		Nodes: []*reader.RichNode{},
	}

	// Conta aninhamento recursivamente através dos pais
	r.buildPathRecursively(node, context, path)

	return path
}

// buildPathRecursively constrói caminho recursivamente pelos pais
func (r *NestedFormsRule) buildPathRecursively(node *reader.RichNode, context map[string]interface{}, path *NestingPath) {
	// Analisa o nó atual
	if r.isTrackedForm(node) {
		formName := r.getFormName(node)
		// Adiciona no início para manter ordem correta (pai → filho)
		path.Forms = append([]string{formName}, path.Forms...)
		path.Nodes = append([]*reader.RichNode{node}, path.Nodes...)
		path.Depth++
	}

	// Processa o pai recursivamente se disponível
	if parent, ok := context["parent"]; ok {
		if parentNode, ok := parent.(*reader.RichNode); ok {
			// Simula contexto do pai (sem o contexto completo, usamos nil para parent do pai)
			parentContext := map[string]interface{}{}
			r.buildPathRecursively(parentNode, parentContext, path)
		}
	}
}

// buildPathFromContext constrói caminho a partir do contexto de nós pais
func (r *NestedFormsRule) buildPathFromContext(path *NestingPath, context map[string]interface{}) {
	// Esta função não é mais necessária com a abordagem recursiva
	// Mantida para compatibilidade, mas vazia
}

// isTrackedForm verifica se o nó é uma forma que deve ser rastreada
func (r *NestedFormsRule) isTrackedForm(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return false
	}

	formName := firstChild.Value
	for _, tracked := range r.TrackedForms {
		if formName == tracked {
			return true
		}
	}

	return false
}

// getFormName extrai o nome da forma do nó
func (r *NestedFormsRule) getFormName(node *reader.RichNode) string {
	if node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol {
		return node.Children[0].Value
	}
	return "unknown"
}

// exceedsLimits verifica se o caminho de aninhamento excede os limites
func (r *NestedFormsRule) exceedsLimits(path *NestingPath) bool {
	if path.Depth <= 1 {
		return false // Não há aninhamento para reportar
	}

	// Verifica profundidade máxima
	if path.Depth > r.MaxDepth {
		return true
	}

	// Verifica se há formas suficientes no caminho para considerar problemático
	if len(path.Forms) >= r.MinFormsInPath {
		return true
	}

	return false
}

// getSuggestionForPath gera sugestões baseadas no padrão de aninhamento
func (r *NestedFormsRule) getSuggestionForPath(path *NestingPath) string {
	if len(path.Forms) == 0 {
		return "Consider flattening the nested structure."
	}

	// Detecta padrões específicos e sugere refatorações
	switch {
	case r.hasPattern(path.Forms, []string{"let", "when", "let"}):
		return "Consider using 'when-let' or 'some->' threading macro to flatten nested let/when forms."
	case r.hasPattern(path.Forms, []string{"let", "if", "let"}):
		return "Consider using 'if-let' or 'some->' threading macro to flatten nested let/if forms."
	case r.countOccurrences(path.Forms, "let") >= 3:
		return "Consider combining multiple 'let' bindings into a single form or using '->' threading macro."
	case r.countOccurrences(path.Forms, "when") >= 2 && r.countOccurrences(path.Forms, "let") >= 1:
		return "Consider using 'when-let', 'some->', or 'and' to flatten nested when/let conditions."
	case r.hasOnlyConditionals(path.Forms):
		return "Consider using 'cond' to flatten nested conditional forms."
	default:
		return fmt.Sprintf("Consider refactoring to reduce nesting depth. Use threading macros (-> or ->>) or combine forms where possible.")
	}
}

// hasPattern verifica se as formas contêm um padrão específico
func (r *NestedFormsRule) hasPattern(forms []string, pattern []string) bool {
	if len(forms) < len(pattern) {
		return false
	}

	for i := 0; i <= len(forms)-len(pattern); i++ {
		match := true
		for j, p := range pattern {
			if forms[i+j] != p {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// countOccurrences conta ocorrências de uma forma específica
func (r *NestedFormsRule) countOccurrences(forms []string, form string) int {
	count := 0
	for _, f := range forms {
		if f == form {
			count++
		}
	}
	return count
}

// hasOnlyConditionals verifica se o caminho contém apenas condicionais
func (r *NestedFormsRule) hasOnlyConditionals(forms []string) bool {
	conditionals := map[string]bool{
		"if": true, "when": true, "if-not": true, "when-not": true,
		"if-let": true, "when-let": true, "if-some": true, "when-some": true,
	}

	for _, form := range forms {
		if !conditionals[form] {
			return false
		}
	}
	return len(forms) >= 2
}

// init registra a regra de nested forms
func init() {
	defaultRule := &NestedFormsRule{
		Rule: Rule{
			ID:          "nested-forms",
			Name:        "Nested Forms",
			Description: "Detects excessive nesting of forms like let, when, if. Deep nesting makes code harder to read and understand. Consider using threading macros, combining forms, or other refactoring techniques to flatten the structure.",
			Severity:    SeverityWarning,
		},
		MaxDepth:       3, // Máximo de 3 níveis de profundidade
		MinFormsInPath: 2, // Mínimo de 2 formas no caminho para reportar
		TrackedForms: []string{
			"let", "when", "if", "when-let", "if-let", "when-some", "if-some",
			"when-not", "if-not", "loop", "binding", "with-open", "with-local-vars",
			"doseq", "dotimes", "for",
		},
	}

	RegisterRule(defaultRule)
}
