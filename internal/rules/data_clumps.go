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
	maxGroups       int
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

var (
	globalDataClumpsAnalyzer     *DataClumpsAnalyzer
	globalDataClumpsAnalyzerOnce sync.Once
)

func GetGlobalDataClumpsAnalyzer() *DataClumpsAnalyzer {
	globalDataClumpsAnalyzerOnce.Do(func() {
		globalDataClumpsAnalyzer = &DataClumpsAnalyzer{
			parameterGroups: make([]ParameterGroup, 0),
			maxGroups:       10000,
		}
	})
	return globalDataClumpsAnalyzer
}

type DataClumpsRule struct {
	Rule
	MinClumpSize        int     `json:"min_clump_size" yaml:"min_clump_size"`
	MinOccurrences      int     `json:"min_occurrences" yaml:"min_occurrences"`
	SimilarityThreshold float64 `json:"similarity_threshold" yaml:"similarity_threshold"`
	MaxParametersCheck  int
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

	if len(analyzer.parameterGroups) < analyzer.maxGroups {
		analyzer.parameterGroups = append(analyzer.parameterGroups, *group)
	}

	analyzer.mu.Unlock()

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

	if len(params) > r.MaxParametersCheck {
		return nil
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

	paramCombinations := r.generateParameterCombinationsOptimized(groups)

	for combination, occurrences := range paramCombinations {
		if len(occurrences) >= r.MinOccurrences {
			similarity := r.calculateSimilarityOptimized(occurrences)
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

func (r *DataClumpsRule) generateParameterCombinationsOptimized(groups []ParameterGroup) map[string][]ParameterGroup {
	combinations := make(map[string][]ParameterGroup)

	for _, group := range groups {

		maxSize := len(group.Parameters)
		if maxSize > r.MaxParametersCheck {
			maxSize = r.MaxParametersCheck
		}

		for size := r.MinClumpSize; size <= maxSize && size <= 6; size++ {
			combos := r.getCombinationsOptimized(group.Parameters, size)
			for _, combo := range combos {

				sort.Strings(combo)
				key := strings.Join(combo, ",")
				combinations[key] = append(combinations[key], group)
			}
		}
	}

	return combinations
}

func (r *DataClumpsRule) getCombinationsOptimized(items []string, k int) [][]string {
	if k > len(items) || k <= 0 || k > 6 {
		return nil
	}

	if len(items) > r.MaxParametersCheck {
		return nil
	}

	if k == 1 {
		result := make([][]string, len(items))
		for i, item := range items {
			result[i] = []string{item}
		}
		return result
	}

	var result [][]string
	for i := 0; i <= len(items)-k; i++ {
		head := items[i]
		tailCombos := r.getCombinationsOptimized(items[i+1:], k-1)
		for _, tailCombo := range tailCombos {
			combo := make([]string, 0, k)
			combo = append(combo, head)
			combo = append(combo, tailCombo...)
			result = append(result, combo)
		}
	}

	return result
}

func (r *DataClumpsRule) calculateSimilarityOptimized(occurrences []ParameterGroup) float64 {
	if len(occurrences) <= 1 {
		return 1.0
	}

	if len(occurrences) > 100 {
		return r.calculateSimilaritySampling(occurrences)
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

func (r *DataClumpsRule) calculateSimilaritySampling(occurrences []ParameterGroup) float64 {

	sample := occurrences
	if len(occurrences) > 50 {
		sample = occurrences[:50]
	}

	return r.calculateSimilarityOptimized(sample)
}

func (r *DataClumpsRule) parametersOverlap(params1, params2 []string) bool {

	if len(params1) < r.MinClumpSize || len(params2) < r.MinClumpSize {
		return false
	}

	set1 := make(map[string]bool, len(params1))
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

func (d *DataClumpsAnalyzer) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.parameterGroups = make([]ParameterGroup, 0)
}

func (d *DataClumpsAnalyzer) GenerateFindings() []*Finding {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.parameterGroups) < 2 {
		return nil
	}

	sort.Slice(d.parameterGroups, func(i, j int) bool {
		if d.parameterGroups[i].Filepath != d.parameterGroups[j].Filepath {
			return d.parameterGroups[i].Filepath < d.parameterGroups[j].Filepath
		}
		if d.parameterGroups[i].FunctionName != d.parameterGroups[j].FunctionName {
			return d.parameterGroups[i].FunctionName < d.parameterGroups[j].FunctionName
		}
		return strings.Join(d.parameterGroups[i].Parameters, ",") < strings.Join(d.parameterGroups[j].Parameters, ",")
	})

	rule := &DataClumpsRule{
		Rule: Rule{
			ID:       "data-clumps",
			Severity: SeverityWarning,
		},
		MinClumpSize:        3,
		MinOccurrences:      2,
		SimilarityThreshold: 0.7,
		MaxParametersCheck:  8,
	}

	clumps := rule.findDataClumps(d.parameterGroups)
	var findings []*Finding

	for _, clump := range clumps {
		for _, occurrence := range clump.Occurrences {
			findings = append(findings, &Finding{
				RuleID:   rule.ID,
				Message:  rule.formatClumpMessage(clump),
				Filepath: occurrence.Filepath,
				Location: occurrence.Location,
				Severity: rule.Severity,
			})
		}
	}

	return findings
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
		MaxParametersCheck:  8,
	}

	RegisterRule(defaultRule)
}
