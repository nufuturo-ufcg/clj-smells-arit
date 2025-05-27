package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
)

type DivergentChangeRule struct {
	Rule
	minResponsibilityThreshold int
	maxComplexityThreshold     int
}

func NewDivergentChangeRule(cfg *config.Config) *DivergentChangeRule {
	rule := &DivergentChangeRule{
		Rule: Rule{
			ID:          "divergent-change",
			Name:        "Divergent Change",
			Description: "Detects functions/namespaces that mix multiple unrelated responsibilities, making them change for different reasons.",
			Severity:    SeverityWarning,
		},
		minResponsibilityThreshold: 2,  // Mínimo de 2 tipos diferentes de responsabilidades
		maxComplexityThreshold:     15, // Limite de complexidade ciclomática
	}

	// Configuração via config se disponível
	if cfg != nil {
		rule.minResponsibilityThreshold = cfg.GetRuleSettingInt("divergent-change", "responsibility-threshold", rule.minResponsibilityThreshold)
		rule.maxComplexityThreshold = cfg.GetRuleSettingInt("divergent-change", "complexity-threshold", rule.maxComplexityThreshold)
	}

	return rule
}

// ResponsibilityType representa diferentes tipos de responsabilidades
type ResponsibilityType int

const (
	ResponsibilityUnknown ResponsibilityType = iota
	ResponsibilityIO
	ResponsibilityDataTransformation
	ResponsibilityValidation
	ResponsibilityBusinessLogic
	ResponsibilityErrorHandling
	ResponsibilityLogging
	ResponsibilityConfiguration
	ResponsibilityNetworking
	ResponsibilityPersistence
)

func (rt ResponsibilityType) String() string {
	switch rt {
	case ResponsibilityUnknown:
		return "Unknown"
	case ResponsibilityIO:
		return "I/O Operations"
	case ResponsibilityDataTransformation:
		return "Data Transformation"
	case ResponsibilityValidation:
		return "Validation"
	case ResponsibilityBusinessLogic:
		return "Business Logic"
	case ResponsibilityErrorHandling:
		return "Error Handling"
	case ResponsibilityLogging:
		return "Logging"
	case ResponsibilityConfiguration:
		return "Configuration"
	case ResponsibilityNetworking:
		return "Networking"
	case ResponsibilityPersistence:
		return "Data Persistence"
	default:
		return "Unknown"
	}
}

type ResponsibilityAnalysis struct {
	responsibilities     map[ResponsibilityType]int
	cyclomaticComplexity int
	totalOperations      int
}

func (r *DivergentChangeRule) Meta() Rule {
	return r.Rule
}

func (r *DivergentChangeRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Analisa tanto funções quanto namespaces
	if r.isFunction(node) {
		return r.checkFunction(node, filepath)
	} else if r.isNamespace(node) {
		return r.checkNamespace(node, filepath)
	}

	return nil
}

func (r *DivergentChangeRule) isFunction(node *reader.RichNode) bool {
	return node.Type == reader.NodeList &&
		len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || node.Children[0].Value == "defn-")
}

func (r *DivergentChangeRule) isNamespace(node *reader.RichNode) bool {
	return node.Type == reader.NodeList &&
		len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		node.Children[0].Value == "ns"
}

func (r *DivergentChangeRule) checkFunction(node *reader.RichNode, filepath string) *Finding {
	if len(node.Children) < 3 {
		return nil
	}

	analysis := r.analyzeResponsibilities(node)

	// Verifica se há múltiplas responsabilidades não relacionadas
	responsibilityCount := len(analysis.responsibilities)

	if responsibilityCount >= r.minResponsibilityThreshold {
		// Verifica se as responsabilidades são realmente divergentes
		if r.areResponsibilitiesDivergent(analysis.responsibilities) {
			responsibilityNames := r.getResponsibilityNames(analysis.responsibilities)

			return &Finding{
				RuleID: r.ID,
				Message: fmt.Sprintf("Function '%s' handles %d different types of responsibilities (%s). Consider separating these concerns into different functions.",
					r.getNodeName(node), responsibilityCount, strings.Join(responsibilityNames, ", ")),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	// Verifica complexidade ciclomática alta como indicador adicional
	if analysis.cyclomaticComplexity > r.maxComplexityThreshold && responsibilityCount >= 2 {
		return &Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf("Function '%s' has high cyclomatic complexity (%d) and multiple responsibilities. This makes it prone to divergent changes.",
				r.getNodeName(node), analysis.cyclomaticComplexity),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func (r *DivergentChangeRule) checkNamespace(node *reader.RichNode, filepath string) *Finding {
	// Para namespaces, verifica se há muitas funções com responsabilidades diferentes
	// Isso pode indicar que o namespace está fazendo muitas coisas diferentes

	// Implementação simplificada - pode ser expandida
	return nil
}

func (r *DivergentChangeRule) analyzeResponsibilities(node *reader.RichNode) *ResponsibilityAnalysis {
	analysis := &ResponsibilityAnalysis{
		responsibilities:     make(map[ResponsibilityType]int),
		cyclomaticComplexity: 1, // Começa com 1
	}

	r.visitNode(node, analysis)

	return analysis
}

func (r *DivergentChangeRule) visitNode(node *reader.RichNode, analysis *ResponsibilityAnalysis) {
	if node == nil {
		return
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		funcName := node.Children[0].Value

		// Identifica tipo de responsabilidade
		if responsibility := r.identifyResponsibility(funcName); responsibility != ResponsibilityUnknown {
			analysis.responsibilities[responsibility]++
			analysis.totalOperations++
		}

		// Calcula complexidade ciclomática
		if r.isDecisionPoint(funcName) {
			analysis.cyclomaticComplexity++
		}
	}

	// Visita recursivamente os filhos
	for _, child := range node.Children {
		r.visitNode(child, analysis)
	}
}

func (r *DivergentChangeRule) identifyResponsibility(funcName string) ResponsibilityType {
	// I/O Operations
	if r.isIOOperation(funcName) {
		return ResponsibilityIO
	}

	// Data Transformation
	if r.isDataTransformation(funcName) {
		return ResponsibilityDataTransformation
	}

	// Validation
	if r.isValidation(funcName) {
		return ResponsibilityValidation
	}

	// Error Handling
	if r.isErrorHandling(funcName) {
		return ResponsibilityErrorHandling
	}

	// Logging
	if r.isLogging(funcName) {
		return ResponsibilityLogging
	}

	// Networking
	if r.isNetworking(funcName) {
		return ResponsibilityNetworking
	}

	// Persistence
	if r.isPersistence(funcName) {
		return ResponsibilityPersistence
	}

	// Configuration
	if r.isConfiguration(funcName) {
		return ResponsibilityConfiguration
	}

	return ResponsibilityUnknown // Não identificado
}

func (r *DivergentChangeRule) isIOOperation(funcName string) bool {
	ioFunctions := map[string]bool{
		"println": true, "print": true, "printf": true,
		"slurp": true, "spit": true,
		"read": true, "read-line": true,
		"with-open": true,
	}
	return ioFunctions[funcName]
}

func (r *DivergentChangeRule) isDataTransformation(funcName string) bool {
	transformFunctions := map[string]bool{
		"map": true, "filter": true, "reduce": true,
		"str": true, "assoc": true, "dissoc": true,
		"get": true, "get-in": true, "update": true,
		"conj": true, "into": true, "merge": true,
		"select-keys": true, "rename-keys": true,
		"group-by": true, "partition": true,
		">=": true, "<=": true, ">": true, "<": true,
		"=": true, "not=": true,
		"inc": true, "dec": true, "+": true, "-": true, "*": true, "/": true,
	}
	return transformFunctions[funcName]
}

func (r *DivergentChangeRule) isValidation(funcName string) bool {
	validationFunctions := map[string]bool{
		"valid?": true, "spec/valid?": true,
		"nil?": true, "empty?": true, "blank?": true,
		"number?": true, "string?": true, "keyword?": true,
		"pos?": true, "neg?": true, "zero?": true,
		"even?": true, "odd?": true,
	}
	return validationFunctions[funcName] || strings.Contains(funcName, "valid")
}

func (r *DivergentChangeRule) isErrorHandling(funcName string) bool {
	errorFunctions := map[string]bool{
		"try": true, "catch": true, "throw": true,
		"ex-info": true, "ex-data": true, "ex-message": true,
		"assert": true,
	}
	return errorFunctions[funcName] || strings.Contains(funcName, "error") || strings.Contains(funcName, "exception")
}

func (r *DivergentChangeRule) isLogging(funcName string) bool {
	loggingFunctions := map[string]bool{
		"log": true, "debug": true, "info": true, "warn": true, "error": true,
		"trace": true,
	}
	return loggingFunctions[funcName] || strings.Contains(funcName, "log")
}

func (r *DivergentChangeRule) isNetworking(funcName string) bool {
	networkFunctions := map[string]bool{
		"http/get": true, "http/post": true, "http/put": true, "http/delete": true,
		"client/get": true, "client/post": true,
		"request": true, "response": true,
	}
	return networkFunctions[funcName] || strings.Contains(funcName, "http") || strings.Contains(funcName, "request")
}

func (r *DivergentChangeRule) isPersistence(funcName string) bool {
	persistenceFunctions := map[string]bool{
		"insert": true, "update": true, "delete": true, "select": true,
		"save": true, "find": true, "query": true,
		"create": true, "drop": true,
	}
	return persistenceFunctions[funcName] || strings.Contains(funcName, "db") || strings.Contains(funcName, "sql")
}

func (r *DivergentChangeRule) isConfiguration(funcName string) bool {
	configFunctions := map[string]bool{
		"config": true, "env": true, "property": true,
		"setting": true, "param": true,
	}
	return configFunctions[funcName] || strings.Contains(funcName, "config") || strings.Contains(funcName, "env")
}

func (r *DivergentChangeRule) isDecisionPoint(funcName string) bool {
	decisionPoints := map[string]bool{
		"if": true, "when": true, "cond": true, "case": true,
		"and": true, "or": true,
		"while": true, "loop": true,
	}
	return decisionPoints[funcName]
}

func (r *DivergentChangeRule) areResponsibilitiesDivergent(responsibilities map[ResponsibilityType]int) bool {
	// Verifica se as responsabilidades são realmente divergentes
	// Combinações problemáticas comuns:

	presentResponsibilities := make([]ResponsibilityType, 0, len(responsibilities))
	for resp := range responsibilities {
		if resp != ResponsibilityUnknown {
			presentResponsibilities = append(presentResponsibilities, resp)
		}
	}

	// Combinações de 2 responsabilidades que são divergentes
	divergentPairs := [][]ResponsibilityType{
		{ResponsibilityIO, ResponsibilityDataTransformation},       // I/O + transformação de dados
		{ResponsibilityIO, ResponsibilityValidation},               // I/O + validação
		{ResponsibilityIO, ResponsibilityPersistence},              // I/O + persistência
		{ResponsibilityNetworking, ResponsibilityLogging},          // Rede + logging
		{ResponsibilityConfiguration, ResponsibilityBusinessLogic}, // Config + lógica de negócio
	}

	// Combinações de 3+ responsabilidades que são sempre divergentes
	divergentCombinations := [][]ResponsibilityType{
		{ResponsibilityIO, ResponsibilityDataTransformation, ResponsibilityValidation},
		{ResponsibilityIO, ResponsibilityBusinessLogic, ResponsibilityPersistence},
		{ResponsibilityNetworking, ResponsibilityDataTransformation, ResponsibilityLogging},
		{ResponsibilityConfiguration, ResponsibilityBusinessLogic, ResponsibilityIO},
	}

	// Verifica combinações de 2 responsabilidades
	if len(presentResponsibilities) == 2 {
		for _, pair := range divergentPairs {
			if r.hasAllResponsibilities(presentResponsibilities, pair) {
				// Para casos de 2 responsabilidades, verifica se há volume significativo
				// Evita falsos positivos em funções muito simples
				totalOps := 0
				for _, count := range responsibilities {
					totalOps += count
				}
				// Só considera divergente se há pelo menos 3 operações no total
				return totalOps >= 3
			}
		}
	}

	// Verifica combinações de 3+ responsabilidades
	if len(presentResponsibilities) >= 3 {
		for _, combination := range divergentCombinations {
			if r.hasAllResponsibilities(presentResponsibilities, combination) {
				return true
			}
		}
		// Se há mais de 3 tipos diferentes, considera divergente
		return len(presentResponsibilities) > 3
	}

	return false
}

func (r *DivergentChangeRule) hasAllResponsibilities(present []ResponsibilityType, required []ResponsibilityType) bool {
	for _, req := range required {
		found := false
		for _, pres := range present {
			if pres == req {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (r *DivergentChangeRule) getResponsibilityNames(responsibilities map[ResponsibilityType]int) []string {
	names := make([]string, 0, len(responsibilities))
	for resp := range responsibilities {
		if resp != ResponsibilityUnknown {
			names = append(names, resp.String())
		}
	}
	return names
}

func (r *DivergentChangeRule) getNodeName(node *reader.RichNode) string {
	if node.Type == reader.NodeList && len(node.Children) > 1 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || node.Children[0].Value == "defn-") {
		if node.Children[1].Type == reader.NodeSymbol {
			return node.Children[1].Value
		}
	}
	return "<unknown_function>"
}

func init() {
	RegisterRule(NewDivergentChangeRule(nil))
}
