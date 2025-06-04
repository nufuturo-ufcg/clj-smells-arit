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
		minResponsibilityThreshold: 2,
		maxComplexityThreshold:     15,
	}

	if cfg != nil {
		rule.minResponsibilityThreshold = cfg.GetRuleSettingInt("divergent-change", "responsibility-threshold", rule.minResponsibilityThreshold)
		rule.maxComplexityThreshold = cfg.GetRuleSettingInt("divergent-change", "complexity-threshold", rule.maxComplexityThreshold)
	}

	return rule
}

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

	responsibilityCount := len(analysis.responsibilities)

	if responsibilityCount >= r.minResponsibilityThreshold {

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

	return nil
}

func (r *DivergentChangeRule) analyzeResponsibilities(node *reader.RichNode) *ResponsibilityAnalysis {
	analysis := &ResponsibilityAnalysis{
		responsibilities:     make(map[ResponsibilityType]int),
		cyclomaticComplexity: 1,
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

		if responsibility := r.identifyResponsibility(funcName); responsibility != ResponsibilityUnknown {
			analysis.responsibilities[responsibility]++
			analysis.totalOperations++
		}

		if r.isDecisionPoint(funcName) {
			analysis.cyclomaticComplexity++
		}
	}

	for _, child := range node.Children {
		r.visitNode(child, analysis)
	}
}

func (r *DivergentChangeRule) identifyResponsibility(funcName string) ResponsibilityType {

	if r.isIOOperation(funcName) {
		return ResponsibilityIO
	}

	if r.isDataTransformation(funcName) {
		return ResponsibilityDataTransformation
	}

	if r.isValidation(funcName) {
		return ResponsibilityValidation
	}

	if r.isErrorHandling(funcName) {
		return ResponsibilityErrorHandling
	}

	if r.isLogging(funcName) {
		return ResponsibilityLogging
	}

	if r.isNetworking(funcName) {
		return ResponsibilityNetworking
	}

	if r.isPersistence(funcName) {
		return ResponsibilityPersistence
	}

	if r.isConfiguration(funcName) {
		return ResponsibilityConfiguration
	}

	return ResponsibilityUnknown
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

	presentResponsibilities := make([]ResponsibilityType, 0, len(responsibilities))
	for resp := range responsibilities {
		if resp != ResponsibilityUnknown {
			presentResponsibilities = append(presentResponsibilities, resp)
		}
	}

	divergentPairs := [][]ResponsibilityType{
		{ResponsibilityIO, ResponsibilityDataTransformation},
		{ResponsibilityIO, ResponsibilityValidation},
		{ResponsibilityIO, ResponsibilityPersistence},
		{ResponsibilityNetworking, ResponsibilityLogging},
		{ResponsibilityConfiguration, ResponsibilityBusinessLogic},
	}

	divergentCombinations := [][]ResponsibilityType{
		{ResponsibilityIO, ResponsibilityDataTransformation, ResponsibilityValidation},
		{ResponsibilityIO, ResponsibilityBusinessLogic, ResponsibilityPersistence},
		{ResponsibilityNetworking, ResponsibilityDataTransformation, ResponsibilityLogging},
		{ResponsibilityConfiguration, ResponsibilityBusinessLogic, ResponsibilityIO},
	}

	if len(presentResponsibilities) == 2 {
		for _, pair := range divergentPairs {
			if r.hasAllResponsibilities(presentResponsibilities, pair) {

				totalOps := 0
				for _, count := range responsibilities {
					totalOps += count
				}

				return totalOps >= 3
			}
		}
	}

	if len(presentResponsibilities) >= 3 {
		for _, combination := range divergentCombinations {
			if r.hasAllResponsibilities(presentResponsibilities, combination) {
				return true
			}
		}

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
