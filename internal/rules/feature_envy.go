// Package rules implementa regra para detectar Feature Envy
// Esta regra identifica funções que fazem mais chamadas para outros namespaces que para o próprio
package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// FeatureEnvyRule detecta funções com inveja de funcionalidades de outros namespaces
// Adaptação do code smell clássico para programação funcional
type FeatureEnvyRule struct {
	Rule
	EnvyThreshold    float64  `json:"envy_threshold" yaml:"envy_threshold"`         // Percentual mínimo de chamadas externas (0.0-1.0)
	MinExternalCalls int      `json:"min_external_calls" yaml:"min_external_calls"` // Mínimo de chamadas externas para considerar
	MinTotalCalls    int      `json:"min_total_calls" yaml:"min_total_calls"`       // Mínimo total de chamadas para analisar
	IgnoreNamespaces []string `json:"ignore_namespaces" yaml:"ignore_namespaces"`   // Namespaces a ignorar (clojure.core, etc)
}

// CallAnalysis representa a análise de chamadas de uma função
type CallAnalysis struct {
	FunctionName    string
	CurrentNS       string
	InternalCalls   int
	ExternalCalls   int
	ExternalTargets map[string]int // namespace -> count
	TotalCalls      int
	EnvyRatio       float64
}

func (r *FeatureEnvyRule) Meta() Rule {
	return r.Rule
}

// Check analisa funções procurando por feature envy
func (r *FeatureEnvyRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Só analisa definições de função
	if !r.isFunctionDefinition(node) {
		return nil
	}

	// Extrai informações da função
	funcName := r.extractFunctionName(node)
	if funcName == "" {
		return nil
	}

	// Determina o namespace atual
	currentNS := r.extractCurrentNamespace(context, filepath)

	// Analisa chamadas dentro da função
	analysis := r.analyzeFunctionCalls(node, currentNS)
	analysis.FunctionName = funcName

	// Verifica se atende aos critérios mínimos
	if !r.meetsMinimumCriteria(analysis) {
		return nil
	}

	// Calcula ratio de inveja
	analysis.EnvyRatio = float64(analysis.ExternalCalls) / float64(analysis.TotalCalls)

	// Verifica se excede o threshold
	if analysis.EnvyRatio >= r.EnvyThreshold {
		return r.createFinding(analysis, filepath, node)
	}

	return nil
}

// isFunctionDefinition verifica se o nó é uma definição de função
func (r *FeatureEnvyRule) isFunctionDefinition(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return false
	}

	return firstChild.Value == "defn" || firstChild.Value == "defn-"
}

// extractFunctionName extrai o nome da função
func (r *FeatureEnvyRule) extractFunctionName(node *reader.RichNode) string {
	if len(node.Children) >= 2 && node.Children[1].Type == reader.NodeSymbol {
		return node.Children[1].Value
	}
	return ""
}

// extractCurrentNamespace extrai o namespace atual do contexto
func (r *FeatureEnvyRule) extractCurrentNamespace(context map[string]interface{}, filepath string) string {
	// TODO: Implementar extração do namespace do contexto
	// Por agora, extraímos do filepath como fallback

	// Tenta extrair do nome do arquivo
	parts := strings.Split(filepath, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		if strings.HasSuffix(filename, ".clj") || strings.HasSuffix(filename, ".cljs") || strings.HasSuffix(filename, ".cljc") {
			// Remove extensão e converte underscores
			nsName := strings.TrimSuffix(filename, ".clj")
			nsName = strings.TrimSuffix(nsName, ".cljs")
			nsName = strings.TrimSuffix(nsName, ".cljc")
			nsName = strings.ReplaceAll(nsName, "_", "-")
			return nsName
		}
	}

	return "unknown"
}

// analyzeFunctionCalls analisa todas as chamadas dentro da função
func (r *FeatureEnvyRule) analyzeFunctionCalls(node *reader.RichNode, currentNS string) *CallAnalysis {
	analysis := &CallAnalysis{
		CurrentNS:       currentNS,
		InternalCalls:   0,
		ExternalCalls:   0,
		ExternalTargets: make(map[string]int),
		TotalCalls:      0,
	}

	r.walkFunctionCalls(node, analysis)

	return analysis
}

// walkFunctionCalls percorre recursivamente o nó procurando por chamadas de função
func (r *FeatureEnvyRule) walkFunctionCalls(node *reader.RichNode, analysis *CallAnalysis) {
	if node == nil {
		return
	}

	// Se é uma lista, pode ser uma chamada de função
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		firstChild := node.Children[0]
		if firstChild.Type == reader.NodeSymbol {
			r.analyzeCall(firstChild.Value, analysis)
		}
	}

	// Continua recursivamente pelos filhos
	for _, child := range node.Children {
		r.walkFunctionCalls(child, analysis)
	}
}

// analyzeCall analisa uma chamada específica
func (r *FeatureEnvyRule) analyzeCall(call string, analysis *CallAnalysis) {
	// Ignora chamadas especiais/macros
	if r.isSpecialForm(call) {
		return
	}

	analysis.TotalCalls++

	// Determina se é chamada externa ou interna
	if strings.Contains(call, "/") {
		// Chamada qualificada (namespace/função)
		parts := strings.SplitN(call, "/", 2)
		namespace := parts[0]

		// Ignora namespaces na lista de ignorados
		if r.shouldIgnoreNamespace(namespace) {
			return
		}

		// É externa se não for o namespace atual
		if namespace != analysis.CurrentNS {
			analysis.ExternalCalls++
			analysis.ExternalTargets[namespace]++
		} else {
			analysis.InternalCalls++
		}
	} else {
		// Chamada não qualificada - assume interna
		analysis.InternalCalls++
	}
}

// isSpecialForm verifica se é uma forma especial que deve ser ignorada
func (r *FeatureEnvyRule) isSpecialForm(call string) bool {
	specialForms := map[string]bool{
		"let": true, "if": true, "when": true, "cond": true, "case": true,
		"do": true, "loop": true, "recur": true, "fn": true, "quote": true,
		"var": true, "def": true, "set!": true, "monitor-enter": true,
		"monitor-exit": true, "throw": true, "try": true, "catch": true,
		"finally": true, "new": true, ".": true,
	}

	return specialForms[call]
}

// shouldIgnoreNamespace verifica se deve ignorar o namespace
func (r *FeatureEnvyRule) shouldIgnoreNamespace(namespace string) bool {
	for _, ignored := range r.IgnoreNamespaces {
		if namespace == ignored {
			return true
		}
	}
	return false
}

// meetsMinimumCriteria verifica se atende aos critérios mínimos para análise
func (r *FeatureEnvyRule) meetsMinimumCriteria(analysis *CallAnalysis) bool {
	if analysis.TotalCalls < r.MinTotalCalls {
		return false
	}

	if analysis.ExternalCalls < r.MinExternalCalls {
		return false
	}

	return true
}

// createFinding cria um finding para feature envy
func (r *FeatureEnvyRule) createFinding(analysis *CallAnalysis, filepath string, node *reader.RichNode) *Finding {
	// Encontra o namespace mais "invejado"
	maxNamespace := ""
	maxCount := 0
	for ns, count := range analysis.ExternalTargets {
		if count > maxCount {
			maxCount = count
			maxNamespace = ns
		}
	}

	suggestion := r.generateSuggestion(analysis, maxNamespace)

	message := fmt.Sprintf(
		"Function '%s' shows feature envy (%.1f%% external calls: %d external vs %d internal). %s",
		analysis.FunctionName,
		analysis.EnvyRatio*100,
		analysis.ExternalCalls,
		analysis.InternalCalls,
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

// generateSuggestion gera sugestão baseada na análise
func (r *FeatureEnvyRule) generateSuggestion(analysis *CallAnalysis, enviedNamespace string) string {
	if enviedNamespace != "" && analysis.ExternalTargets[enviedNamespace] > analysis.InternalCalls {
		return fmt.Sprintf(
			"Consider moving this function to namespace '%s' where it primarily operates, or refactor to reduce external dependencies.",
			enviedNamespace,
		)
	}

	return "Consider refactoring to reduce dependencies on external namespaces, or move this function to where its data/operations primarily reside."
}

// init registra a regra de feature envy
func init() {
	defaultRule := &FeatureEnvyRule{
		Rule: Rule{
			ID:          "feature-envy",
			Name:        "Feature Envy",
			Description: "Detects functions that make more calls to other namespaces than to their own, indicating they might be in the wrong place. Based on the classic 'Feature Envy' code smell adapted for functional programming.",
			Severity:    SeverityWarning,
		},
		EnvyThreshold:    0.7, // 70% de chamadas externas
		MinExternalCalls: 3,   // Mínimo 3 chamadas externas
		MinTotalCalls:    5,   // Mínimo 5 chamadas totais
		IgnoreNamespaces: []string{
			"clojure.core", "cljs.core", "clojure.string", "clojure.set",
			"clojure.walk", "clojure.zip", "clojure.data", "clojure.edn",
			"clojure.java.io", "clojure.pprint", "clojure.repl",
		},
	}

	RegisterRule(defaultRule)
}
