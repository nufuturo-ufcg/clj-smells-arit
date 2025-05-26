// Package rules implementa regras para detectar verificações verbosas em Clojure
// Esta regra específica identifica verificações manuais que podem ser substituídas por funções idiomáticas
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// VerboseChecksRule detecta verificações verbosas que podem ser simplificadas
// Identifica padrões como (= 0 x) que podem ser (zero? x), (= true x) que pode ser (true? x), etc.
type VerboseChecksRule struct {
	Rule
	CheckNumericComparisons bool `json:"check_numeric_comparisons" yaml:"check_numeric_comparisons"` // Verificar comparações numéricas verbosas
	CheckBooleanComparisons bool `json:"check_boolean_comparisons" yaml:"check_boolean_comparisons"` // Verificar comparações booleanas verbosas
	CheckNilComparisons     bool `json:"check_nil_comparisons" yaml:"check_nil_comparisons"`         // Verificar comparações com nil verbosas
	CheckMathOperations     bool `json:"check_math_operations" yaml:"check_math_operations"`         // Verificar operações matemáticas verbosas
}

func (r *VerboseChecksRule) Meta() Rule {
	return r.Rule
}

// numericComparisons mapeia comparações verbosas para suas versões idiomáticas
var numericComparisons = map[string]map[string]string{
	"=": {
		"0": "zero?",
	},
	">": {
		"0": "pos?", // (> x 0) -> (pos? x)
	},
	"<": {
		"0": "neg?", // (< x 0) -> (neg? x)
	},
}

// booleanComparisons mapeia comparações booleanas verbosas
var booleanComparisons = map[string]string{
	"true":  "true?",
	"false": "false?",
}

// mathOperations mapeia operações matemáticas verbosas
var mathOperations = map[string]map[string]string{
	"+": {
		"1": "inc", // (+ 1 x) ou (+ x 1) -> (inc x)
	},
	"-": {
		"1": "dec", // (- x 1) -> (dec x)
	},
}

// detectNumericComparison detecta comparações numéricas verbosas
func (r *VerboseChecksRule) detectNumericComparison(node *reader.RichNode) *Finding {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol {
		return nil
	}

	operator := opNode.Value
	comparisons, exists := numericComparisons[operator]
	if !exists {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	// Verifica padrões como (= 0 x), (> x 0), (< x 0)
	var constantValue, variableExpr string
	var suggestion string

	if arg1.Type == reader.NodeNumber {
		constantValue = arg1.Value
		variableExpr = getVerboseNodeText(arg2)
		if idiomaticFunc, exists := comparisons[constantValue]; exists {
			if operator == "=" {
				suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			} else if operator == ">" && constantValue == "0" {
				// (> 0 x) -> (neg? x) - inverte a lógica
				suggestion = fmt.Sprintf("(neg? %s)", variableExpr)
			} else if operator == "<" && constantValue == "0" {
				// (< 0 x) -> (pos? x) - inverte a lógica
				suggestion = fmt.Sprintf("(pos? %s)", variableExpr)
			}
		}
	} else if arg2.Type == reader.NodeNumber {
		constantValue = arg2.Value
		variableExpr = getVerboseNodeText(arg1)
		if idiomaticFunc, exists := comparisons[constantValue]; exists {
			suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
		}
	}

	if suggestion != "" {
		originalExpr := fmt.Sprintf("(%s %s %s)", operator, getVerboseNodeText(arg1), getVerboseNodeText(arg2))
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Verbose numeric comparison: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
			Filepath: "", // será preenchido pelo caller
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

// detectBooleanComparison detecta comparações booleanas verbosas
func (r *VerboseChecksRule) detectBooleanComparison(node *reader.RichNode) *Finding {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol || opNode.Value != "=" {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	var constantValue, variableExpr string
	var suggestion string

	// Verifica padrões como (= true x) ou (= x true)
	if arg1.Type == reader.NodeSymbol && (arg1.Value == "true" || arg1.Value == "false") {
		constantValue = arg1.Value
		variableExpr = getVerboseNodeText(arg2)
	} else if arg2.Type == reader.NodeSymbol && (arg2.Value == "true" || arg2.Value == "false") {
		constantValue = arg2.Value
		variableExpr = getVerboseNodeText(arg1)
	}

	if constantValue != "" {
		if idiomaticFunc, exists := booleanComparisons[constantValue]; exists {
			suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			originalExpr := fmt.Sprintf("(= %s %s)", getVerboseNodeText(arg1), getVerboseNodeText(arg2))
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Verbose boolean comparison: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
				Filepath: "", // será preenchido pelo caller
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

// detectNilComparison detecta comparações com nil verbosas
func (r *VerboseChecksRule) detectNilComparison(node *reader.RichNode) *Finding {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol || opNode.Value != "=" {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	var variableExpr string
	var isNilComparison bool

	// Verifica padrões como (= nil x) ou (= x nil)
	if arg1.Type == reader.NodeSymbol && arg1.Value == "nil" {
		variableExpr = getVerboseNodeText(arg2)
		isNilComparison = true
	} else if arg2.Type == reader.NodeSymbol && arg2.Value == "nil" {
		variableExpr = getVerboseNodeText(arg1)
		isNilComparison = true
	}

	if isNilComparison {
		suggestion := fmt.Sprintf("(nil? %s)", variableExpr)
		originalExpr := fmt.Sprintf("(= %s %s)", getVerboseNodeText(arg1), getVerboseNodeText(arg2))
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Verbose nil comparison: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
			Filepath: "", // será preenchido pelo caller
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

// detectMathOperation detecta operações matemáticas verbosas
func (r *VerboseChecksRule) detectMathOperation(node *reader.RichNode) *Finding {
	if node.Type != reader.NodeList || len(node.Children) != 3 {
		return nil
	}

	opNode := node.Children[0]
	if opNode.Type != reader.NodeSymbol {
		return nil
	}

	operator := opNode.Value
	operations, exists := mathOperations[operator]
	if !exists {
		return nil
	}

	arg1 := node.Children[1]
	arg2 := node.Children[2]

	var constantValue, variableExpr string
	var suggestion string

	// Para adição, aceita tanto (+ 1 x) quanto (+ x 1)
	if operator == "+" {
		if arg1.Type == reader.NodeNumber && arg1.Value == "1" {
			constantValue = arg1.Value
			variableExpr = getVerboseNodeText(arg2)
		} else if arg2.Type == reader.NodeNumber && arg2.Value == "1" {
			constantValue = arg2.Value
			variableExpr = getVerboseNodeText(arg1)
		}
	} else if operator == "-" {
		// Para subtração, apenas (- x 1) -> (dec x)
		if arg2.Type == reader.NodeNumber && arg2.Value == "1" {
			constantValue = arg2.Value
			variableExpr = getVerboseNodeText(arg1)
		}
	}

	if constantValue != "" {
		if idiomaticFunc, exists := operations[constantValue]; exists {
			suggestion = fmt.Sprintf("(%s %s)", idiomaticFunc, variableExpr)
			originalExpr := fmt.Sprintf("(%s %s %s)", operator, getVerboseNodeText(arg1), getVerboseNodeText(arg2))
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Verbose math operation: `%s`. Consider using the more idiomatic `%s`.", originalExpr, suggestion),
				Filepath: "", // será preenchido pelo caller
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

// getVerboseNodeText extrai texto representativo de um nó para verbose checks
func getVerboseNodeText(node *reader.RichNode) string {
	if node == nil {
		return "nil"
	}

	switch node.Type {
	case reader.NodeSymbol, reader.NodeKeyword, reader.NodeString, reader.NodeNumber:
		return node.Value
	case reader.NodeList:
		if len(node.Children) > 0 {
			return "(" + getVerboseNodeText(node.Children[0]) + " ...)"
		}
		return "()"
	case reader.NodeVector:
		return "[...]"
	case reader.NodeMap:
		return "{...}"
	case reader.NodeSet:
		return "#{...}"
	default:
		return "..."
	}
}

// Check analisa nós procurando por verificações verbosas
func (r *VerboseChecksRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Verifica se é uma lista com pelo menos 3 elementos (operador e 2 argumentos)
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	// 1. Detecta comparações numéricas verbosas
	if r.CheckNumericComparisons {
		if finding := r.detectNumericComparison(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	// 2. Detecta comparações booleanas verbosas
	if r.CheckBooleanComparisons {
		if finding := r.detectBooleanComparison(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	// 3. Detecta comparações com nil verbosas
	if r.CheckNilComparisons {
		if finding := r.detectNilComparison(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	// 4. Detecta operações matemáticas verbosas
	if r.CheckMathOperations {
		if finding := r.detectMathOperation(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	return nil
}

// init registra a regra de Verbose Checks com configurações padrão
func init() {
	defaultRule := &VerboseChecksRule{
		Rule: Rule{
			ID:          "verbose-checks",
			Name:        "Verbose Checks",
			Description: "Detects verbose checks that can be simplified using idiomatic Clojure functions. This includes manual implementations of common checks like (= 0 x) instead of (zero? x), (= true x) instead of (true? x), (+ 1 x) instead of (inc x), and similar patterns. Based on idiomatic Clojure practices from bsless.github.io/code-smells.",
			Severity:    SeverityHint,
		},
		CheckNumericComparisons: true, // Verificar comparações numéricas por padrão
		CheckBooleanComparisons: true, // Verificar comparações booleanas por padrão
		CheckNilComparisons:     true, // Verificar comparações com nil por padrão
		CheckMathOperations:     true, // Verificar operações matemáticas por padrão
	}

	RegisterRule(defaultRule)
}
