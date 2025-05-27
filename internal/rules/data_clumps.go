// Package rules implementa regras para detectar data clumps em código Clojure
// Esta regra específica identifica grupos de parâmetros que aparecem frequentemente juntos
package rules

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/thlaurentino/arit/internal/reader"
)

// DataClumpsAnalyzer é um analisador global para data clumps
type DataClumpsAnalyzer struct {
	mu              sync.Mutex
	parameterGroups []ParameterGroup
}

// ParameterGroup representa um grupo de parâmetros em uma função
type ParameterGroup struct {
	FunctionName string
	Parameters   []string
	Location     *reader.Location
	Filepath     string
}

// ClumpCandidate representa um candidato a data clump
type ClumpCandidate struct {
	Parameters  []string
	Occurrences []ParameterGroup
	Similarity  float64
}

var globalDataClumpsAnalyzer *DataClumpsAnalyzer

// GetGlobalDataClumpsAnalyzer retorna a instância global do analisador
func GetGlobalDataClumpsAnalyzer() *DataClumpsAnalyzer {
	if globalDataClumpsAnalyzer == nil {
		globalDataClumpsAnalyzer = &DataClumpsAnalyzer{
			parameterGroups: make([]ParameterGroup, 0),
		}
	}
	return globalDataClumpsAnalyzer
}

// DataClumpsRule detecta grupos de dados que aparecem frequentemente juntos
// Identifica parâmetros que sempre aparecem em conjunto e sugere agrupamento
type DataClumpsRule struct {
	Rule
	MinClumpSize        int     `json:"min_clump_size" yaml:"min_clump_size"`             // Tamanho mínimo do clump
	MinOccurrences      int     `json:"min_occurrences" yaml:"min_occurrences"`           // Mínimo de ocorrências para considerar clump
	SimilarityThreshold float64 `json:"similarity_threshold" yaml:"similarity_threshold"` // Limiar de similaridade (0.0-1.0)
}

func (r *DataClumpsRule) Meta() Rule {
	return r.Rule
}

// Check analisa funções procurando por data clumps
// Coleta parâmetros de função e adiciona ao analisador global
func (r *DataClumpsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Só processa definições de função
	if !r.isFunctionDefinition(node) {
		return nil
	}

	// Extrai grupo de parâmetros da função
	group := r.extractParameterGroup(node, filepath)
	if group == nil {
		return nil
	}

	// Adiciona ao analisador global
	analyzer := GetGlobalDataClumpsAnalyzer()
	analyzer.mu.Lock()
	analyzer.parameterGroups = append(analyzer.parameterGroups, *group)

	// Verifica se temos grupos suficientes para análise
	if len(analyzer.parameterGroups) >= r.MinOccurrences {
		clumps := r.findDataClumps(analyzer.parameterGroups)
		analyzer.mu.Unlock()

		// Retorna o primeiro clump encontrado que inclui esta função
		for _, clump := range clumps {
			for _, occurrence := range clump.Occurrences {
				if occurrence.FunctionName == group.FunctionName && occurrence.Filepath == filepath {
					return &Finding{
						RuleID:   r.ID,
						Message:  r.formatClumpMessage(clump),
						Filepath: filepath,
						Location: occurrence.Location,
						Severity: r.Severity,
					}
				}
			}
		}
	} else {
		analyzer.mu.Unlock()
	}

	return nil
}

// isFunctionDefinition verifica se é uma definição de função
func (r *DataClumpsRule) isFunctionDefinition(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return false
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return false
	}

	// Aceita diferentes tipos de definições de função
	fnType := firstChild.Value
	return fnType == "defn" || fnType == "defn-" || fnType == "defmacro"
}

// extractParameterGroup extrai grupo de parâmetros de uma definição de função
func (r *DataClumpsRule) extractParameterGroup(node *reader.RichNode, filepath string) *ParameterGroup {
	// Verifica se é uma definição de função (defn, defn-, defmacro)
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return nil
	}

	// Aceita diferentes tipos de definições de função
	fnType := firstChild.Value
	if fnType != "defn" && fnType != "defn-" && fnType != "defmacro" {
		return nil
	}

	// Extrai nome da função
	funcNameNode := node.Children[1]
	if funcNameNode.Type != reader.NodeSymbol {
		return nil
	}
	funcName := funcNameNode.Value

	// Localiza o vetor de parâmetros (pode ter docstring/metadata)
	var paramsNode *reader.RichNode
	for i := 2; i < len(node.Children); i++ {
		child := node.Children[i]
		if child.Type == reader.NodeVector {
			paramsNode = child
			break
		}
		// Para funções multi-arity, pega a primeira aridade
		if child.Type == reader.NodeList && len(child.Children) > 0 {
			if child.Children[0].Type == reader.NodeVector {
				paramsNode = child.Children[0]
				break
			}
		}
	}

	if paramsNode == nil {
		return nil
	}

	// Extrai nomes dos parâmetros, filtrando especiais
	var params []string
	for _, paramNode := range paramsNode.Children {
		if paramNode.Type == reader.NodeSymbol {
			paramName := paramNode.Value
			// Filtra parâmetros especiais
			if paramName != "&" && !strings.HasPrefix(paramName, "&") && paramName != "_" {
				params = append(params, paramName)
			}
		}
	}

	// Só considera funções com parâmetros suficientes
	if len(params) < r.MinClumpSize {
		return nil
	}

	return &ParameterGroup{
		FunctionName: funcName,
		Parameters:   params,
		Location:     paramsNode.Location,
		Filepath:     filepath,
	}
}

// findDataClumps analisa os grupos de parâmetros procurando por clumps
func (r *DataClumpsRule) findDataClumps(groups []ParameterGroup) []ClumpCandidate {
	var clumps []ClumpCandidate

	// Gera todas as combinações possíveis de parâmetros
	paramCombinations := r.generateParameterCombinations(groups)

	// Analisa cada combinação
	for combination, occurrences := range paramCombinations {
		if len(occurrences) >= r.MinOccurrences {
			similarity := r.calculateSimilarity(occurrences)
			if similarity >= r.SimilarityThreshold {
				// Converte a string de combinação de volta para slice
				params := strings.Split(combination, ",")
				clumps = append(clumps, ClumpCandidate{
					Parameters:  params,
					Occurrences: occurrences,
					Similarity:  similarity,
				})
			}
		}
	}

	// Ordena por número de ocorrências (mais frequentes primeiro)
	sort.Slice(clumps, func(i, j int) bool {
		return len(clumps[i].Occurrences) > len(clumps[j].Occurrences)
	})

	return clumps
}

// generateParameterCombinations gera todas as combinações de parâmetros
func (r *DataClumpsRule) generateParameterCombinations(groups []ParameterGroup) map[string][]ParameterGroup {
	combinations := make(map[string][]ParameterGroup)

	for _, group := range groups {
		// Gera combinações de tamanho MinClumpSize ou maior
		for size := r.MinClumpSize; size <= len(group.Parameters); size++ {
			combos := r.getCombinations(group.Parameters, size)
			for _, combo := range combos {
				// Ordena para normalizar a chave
				sort.Strings(combo)
				key := strings.Join(combo, ",")
				combinations[key] = append(combinations[key], group)
			}
		}
	}

	return combinations
}

// getCombinations gera todas as combinações de tamanho k de uma slice
func (r *DataClumpsRule) getCombinations(items []string, k int) [][]string {
	if k > len(items) || k <= 0 {
		return nil
	}

	if k == 1 {
		var result [][]string
		for _, item := range items {
			result = append(result, []string{item})
		}
		return result
	}

	var result [][]string
	for i := 0; i <= len(items)-k; i++ {
		head := items[i]
		tailCombos := r.getCombinations(items[i+1:], k-1)
		for _, tailCombo := range tailCombos {
			combo := append([]string{head}, tailCombo...)
			result = append(result, combo)
		}
	}

	return result
}

// calculateSimilarity calcula a similaridade entre ocorrências de um clump
func (r *DataClumpsRule) calculateSimilarity(occurrences []ParameterGroup) float64 {
	if len(occurrences) <= 1 {
		return 1.0
	}

	// Calcula similaridade baseada na consistência dos parâmetros
	// e na frequência de aparição conjunta
	totalPairs := 0
	matchingPairs := 0

	for i := 0; i < len(occurrences); i++ {
		for j := i + 1; j < len(occurrences); j++ {
			totalPairs++
			if r.parametersOverlap(occurrences[i].Parameters, occurrences[j].Parameters) {
				matchingPairs++
			}
		}
	}

	if totalPairs == 0 {
		return 1.0
	}

	return float64(matchingPairs) / float64(totalPairs)
}

// parametersOverlap verifica se dois grupos de parâmetros têm sobreposição significativa
func (r *DataClumpsRule) parametersOverlap(params1, params2 []string) bool {
	set1 := make(map[string]bool)
	for _, p := range params1 {
		set1[p] = true
	}

	overlap := 0
	for _, p := range params2 {
		if set1[p] {
			overlap++
		}
	}

	// Considera sobreposição se pelo menos MinClumpSize parâmetros coincidem
	return overlap >= r.MinClumpSize
}

// formatClumpMessage formata a mensagem do finding
func (r *DataClumpsRule) formatClumpMessage(clump ClumpCandidate) string {
	paramStr := strings.Join(clump.Parameters, ", ")
	functionNames := make([]string, len(clump.Occurrences))
	for i, occ := range clump.Occurrences {
		functionNames[i] = occ.FunctionName
	}

	return fmt.Sprintf(
		"Data clump detected: parameters [%s] appear together in %d functions (%s). Consider grouping them into a map or record.",
		paramStr,
		len(clump.Occurrences),
		strings.Join(functionNames, ", "),
	)
}

// init registra a regra de data clumps com configurações padrão
func init() {
	defaultRule := &DataClumpsRule{
		Rule: Rule{
			ID:          "data-clumps",
			Name:        "Data Clumps",
			Description: "Detects groups of data that frequently appear together in function parameters. These clumps should be turned into their own classes or maps to improve code organization and reduce parameter lists.",
			Severity:    SeverityWarning,
		},
		MinClumpSize:        3,   // Mínimo de 3 parâmetros para formar um clump
		MinOccurrences:      2,   // Deve aparecer em pelo menos 2 funções
		SimilarityThreshold: 0.7, // 70% de similaridade entre ocorrências
	}

	RegisterRule(defaultRule)
}
