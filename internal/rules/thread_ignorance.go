// Package rules implementa regras para detectar ignorância de threading macros em Clojure
// Esta regra específica identifica código que seria mais legível usando -> ou ->> ao invés de aninhamento
package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// ThreadIgnoranceRule detecta código que deveria usar threading macros
// Threading macros como -> e ->> melhoram significativamente a legibilidade de transformações sequenciais
type ThreadIgnoranceRule struct {
	Rule
	MinNestingDepth int `json:"min_nesting_depth" yaml:"min_nesting_depth"` // Profundidade mínima para sugerir threading
	MaxArguments    int `json:"max_arguments" yaml:"max_arguments"`         // Máximo de argumentos para considerar threading
}

func (r *ThreadIgnoranceRule) Meta() Rule {
	return r.Rule
}

// threadingCandidateFunctions define funções que são boas candidatas para threading
// Estas funções geralmente transformam dados e se beneficiam de threading macros
var threadingCandidateFunctions = map[string]bool{
	// Transformações de coleções
	"map":        true,
	"filter":     true,
	"remove":     true,
	"reduce":     true,
	"mapcat":     true,
	"keep":       true,
	"distinct":   true,
	"sort":       true,
	"sort-by":    true,
	"group-by":   true,
	"partition":  true,
	"take":       true,
	"drop":       true,
	"take-while": true,
	"drop-while": true,

	// Operações em mapas
	"assoc":       true,
	"dissoc":      true,
	"update":      true,
	"merge":       true,
	"select-keys": true,

	// Operações em strings
	"str/replace":    true,
	"str/trim":       true,
	"str/upper-case": true,
	"str/lower-case": true,
	"str/split":      true,

	// Conversões
	"vec":  true,
	"set":  true,
	"seq":  true,
	"into": true,
}

// isThreadingCandidate verifica se uma função é candidata para threading
func isThreadingCandidate(funcName string) bool {
	// Verifica nome direto
	if threadingCandidateFunctions[funcName] {
		return true
	}

	// Verifica com namespace (ex: clojure.string/replace)
	parts := strings.Split(funcName, "/")
	if len(parts) == 2 {
		shortName := parts[1]
		return threadingCandidateFunctions[shortName] || threadingCandidateFunctions[funcName]
	}

	return false
}

// countNestedCalls conta o número de chamadas aninhadas em uma expressão
func countNestedCalls(node *reader.RichNode, depth int) int {
	if node == nil || depth > 10 { // Evita recursão infinita
		return 0
	}

	count := 0

	// Se é uma lista que representa uma chamada de função
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		if funcNode := node.Children[0]; funcNode.Type == reader.NodeSymbol {
			if isThreadingCandidate(funcNode.Value) {
				count = 1 // Esta é uma chamada threading candidate
			}
		}

		// Conta recursivamente nos argumentos
		for i := 1; i < len(node.Children); i++ {
			count += countNestedCalls(node.Children[i], depth+1)
		}
	} else {
		// Para outros tipos de nós, verifica os filhos
		for _, child := range node.Children {
			count += countNestedCalls(child, depth+1)
		}
	}

	return count
}

// hasNestedThreadingCandidates verifica se há chamadas aninhadas de funções candidatas a threading
func hasNestedThreadingCandidates(node *reader.RichNode) (bool, int, string) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false, 0, ""
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return false, 0, ""
	}

	funcName := funcNode.Value

	// Verifica se a função principal é candidata a threading
	if !isThreadingCandidate(funcName) {
		return false, 0, ""
	}

	// Conta o total de chamadas threading candidates aninhadas
	totalCalls := countNestedCalls(node, 0)

	// Se há pelo menos 2 chamadas threading candidates (incluindo a atual), sugere threading
	if totalCalls >= 2 {
		// Determina o tipo de threading baseado na função
		threadingType := "thread-first"
		if isCollectionFunction(funcName) {
			threadingType = "thread-last"
		}

		return true, totalCalls, threadingType
	}

	return false, totalCalls, ""
}

// isCollectionFunction verifica se uma função tipicamente opera em coleções (candidata a ->>)
func isCollectionFunction(funcName string) bool {
	collectionFunctions := map[string]bool{
		"map":        true,
		"filter":     true,
		"remove":     true,
		"mapcat":     true,
		"keep":       true,
		"distinct":   true,
		"sort":       true,
		"sort-by":    true,
		"group-by":   true,
		"partition":  true,
		"take":       true,
		"drop":       true,
		"take-while": true,
		"drop-while": true,
		"vec":        true,
		"set":        true,
		"seq":        true,
		"into":       true,
	}

	return collectionFunctions[funcName]
}

// generateThreadingExample gera um exemplo de como o código poderia ser refatorado
func generateThreadingExample(node *reader.RichNode, threadingType string) string {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return ""
	}

	// Extrai a estrutura básica para o exemplo
	funcName := ""
	if funcNode := node.Children[0]; funcNode.Type == reader.NodeSymbol {
		funcName = funcNode.Value
	}

	switch threadingType {
	case "thread-first":
		return fmt.Sprintf("Consider using -> macro: (-> data (%s ...) (next-fn ...))", funcName)
	case "thread-last":
		return fmt.Sprintf("Consider using ->> macro: (->> data (%s ...) (next-fn ...))", funcName)
	default:
		return "Consider using threading macros (-> or ->>) for better readability"
	}
}

// Check analisa nós procurando por padrões que se beneficiariam de threading macros
func (r *ThreadIgnoranceRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Ignora se já está usando threading macros
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		if funcNode := node.Children[0]; funcNode.Type == reader.NodeSymbol {
			funcName := funcNode.Value
			if funcName == "->" || funcName == "->>" || funcName == "some->" || funcName == "as->" || funcName == "cond->" || funcName == "cond->>" {
				return nil // Já está usando threading
			}
		}
	}

	// Verifica se há padrões de aninhamento que se beneficiariam de threading
	hasNested, callCount, threadingType := hasNestedThreadingCandidates(node)

	if !hasNested || callCount < r.MinNestingDepth {
		return nil
	}

	// Verifica se o número de argumentos não é excessivo
	if node.Type == reader.NodeList && len(node.Children) > r.MaxArguments {
		return nil // Muitos argumentos podem não se beneficiar de threading
	}

	example := generateThreadingExample(node, threadingType)

	meta := r.Meta()
	return &Finding{
		RuleID: meta.ID,
		Message: fmt.Sprintf("Nested function calls detected (%d threading candidates). %s. Threading macros improve readability by eliminating nested parentheses and making data flow more explicit.",
			callCount, example),
		Filepath: filepath,
		Location: node.Location,
		Severity: meta.Severity,
	}
}

// init registra a regra de Thread Ignorance com configurações padrão
func init() {
	defaultRule := &ThreadIgnoranceRule{
		Rule: Rule{
			ID:          "thread-ignorance",
			Name:        "Thread Ignorance",
			Description: "Detects nested function calls that would benefit from threading macros (-> or ->>). Threading macros improve readability by making data transformations more explicit and reducing nested parentheses. This rule suggests using -> for 'thread-first' patterns and ->> for 'thread-last' patterns.",
			Severity:    SeverityHint,
		},
		MinNestingDepth: 2, // Mínimo de 2 chamadas threading candidates para sugerir threading
		MaxArguments:    8, // Máximo de 8 argumentos para considerar threading benéfico
	}

	RegisterRule(defaultRule)
}
