package rules

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/thlaurentino/arit/internal/reader"
)

type DataClumpsAnalyzer struct {
	mu              sync.Mutex
	parameterGroups []ParameterGroup
}

type ParameterGroup struct {
	FunctionName string
	Parameters   []string
	Location     *reader.Location
	Filepath     string
}

type ClumpCandidate struct {
	Parameters  []string
	Occurrences []ParameterGroup
	Similarity  float64
}

var globalDataClumpsAnalyzer *DataClumpsAnalyzer

func GetGlobalDataClumpsAnalyzer() *DataClumpsAnalyzer {
	if globalDataClumpsAnalyzer == nil {
		globalDataClumpsAnalyzer = &DataClumpsAnalyzer{
			parameterGroups: make([]ParameterGroup, 0),
		}
	}
	return globalDataClumpsAnalyzer
}

type DataClumpsRule struct {
	Rule
	MinClumpSize        int     `json:"min_clump_size" yaml:"min_clump_size"`
	MinOccurrences      int     `json:"min_occurrences" yaml:"min_occurrences"`
	SimilarityThreshold float64 `json:"similarity_threshold" yaml:"similarity_threshold"`
}

func (r *DataClumpsRule) Meta() Rule {
	return r.Rule
}

func (r *DataClumpsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if !r.isFunctionDefinition(node) {
		return nil
	}

	group := r.extractParameterGroup(node, filepath)
	if group == nil {
		return nil
	}

	analyzer := GetGlobalDataClumpsAnalyzer()
	analyzer.mu.Lock()
	analyzer.parameterGroups = append(analyzer.parameterGroups, *group)

	if len(analyzer.parameterGroups) >= r.MinOccurrences {
		clumps := r.findDataClumps(analyzer.parameterGroups)
		analyzer.mu.Unlock()

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

func (r *DataClumpsRule) isFunctionDefinition(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return false
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return false
	}

	fnType := firstChild.Value
	return fnType == "defn" || fnType == "defn-" || fnType == "defmacro"
}

func (r *DataClumpsRule) extractParameterGroup(node *reader.RichNode, filepath string) *ParameterGroup {

	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return nil
	}

	fnType := firstChild.Value
	if fnType != "defn" && fnType != "defn-" && fnType != "defmacro" {
		return nil
	}

	funcNameNode := node.Children[1]
	if funcNameNode.Type != reader.NodeSymbol {
		return nil
	}
	funcName := funcNameNode.Value

	var paramsNode *reader.RichNode
	for i := 2; i < len(node.Children); i++ {
		child := node.Children[i]
		if child.Type == reader.NodeVector {
			paramsNode = child
			break
		}

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

	var params []string
	for _, paramNode := range paramsNode.Children {
		if paramNode.Type == reader.NodeSymbol {
			paramName := paramNode.Value

			if paramName != "&" && !strings.HasPrefix(paramName, "&") && paramName != "_" {
				params = append(params, paramName)
			}
		}
	}

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

func (r *DataClumpsRule) findDataClumps(groups []ParameterGroup) []ClumpCandidate {
	var clumps []ClumpCandidate

	paramCombinations := r.generateParameterCombinations(groups)

	for combination, occurrences := range paramCombinations {
		if len(occurrences) >= r.MinOccurrences {
			similarity := r.calculateSimilarity(occurrences)
			if similarity >= r.SimilarityThreshold {

				params := strings.Split(combination, ",")
				clumps = append(clumps, ClumpCandidate{
					Parameters:  params,
					Occurrences: occurrences,
					Similarity:  similarity,
				})
			}
		}
	}

	sort.Slice(clumps, func(i, j int) bool {
		return len(clumps[i].Occurrences) > len(clumps[j].Occurrences)
	})

	return clumps
}

func (r *DataClumpsRule) generateParameterCombinations(groups []ParameterGroup) map[string][]ParameterGroup {
	combinations := make(map[string][]ParameterGroup)

	for _, group := range groups {

		for size := r.MinClumpSize; size <= len(group.Parameters); size++ {
			combos := r.getCombinations(group.Parameters, size)
			for _, combo := range combos {

				sort.Strings(combo)
				key := strings.Join(combo, ",")
				combinations[key] = append(combinations[key], group)
			}
		}
	}

	return combinations
}

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

func (r *DataClumpsRule) calculateSimilarity(occurrences []ParameterGroup) float64 {
	if len(occurrences) <= 1 {
		return 1.0
	}

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

	return overlap >= r.MinClumpSize
}

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

func init() {
	defaultRule := &DataClumpsRule{
		Rule: Rule{
			ID:          "data-clumps",
			Name:        "Data Clumps",
			Description: "Detects groups of data that frequently appear together in function parameters. These clumps should be turned into their own classes or maps to improve code organization and reduce parameter lists.",
			Severity:    SeverityWarning,
		},
		MinClumpSize:        3,
		MinOccurrences:      2,
		SimilarityThreshold: 0.7,
	}

	RegisterRule(defaultRule)
}
