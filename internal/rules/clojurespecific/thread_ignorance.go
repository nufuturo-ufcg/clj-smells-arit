package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type ThreadIgnoranceRule struct {
	rules.Rule
	MinNestingDepth int `json:"min_nesting_depth" yaml:"min_nesting_depth"`
	MinLetChain     int `json:"min_let_chain" yaml:"min_let_chain"`
}

func (r *ThreadIgnoranceRule) Meta() rules.Rule {
	return r.Rule
}

var threadingCandidateFunctions = map[string]bool{
	"map": true, "filter": true, "remove": true, "reduce": true, "mapcat": true,
	"keep": true, "distinct": true, "sort": true, "sort-by": true, "group-by": true,
	"partition": true, "take": true, "drop": true, "take-while": true, "drop-while": true,
	"assoc": true, "dissoc": true, "update": true, "merge": true, "select-keys": true,
	"str/replace": true, "str/trim": true, "str/upper-case": true, "str/lower-case": true, "str/split": true,
	"vec": true, "set": true, "seq": true, "into": true,
	// Adicionados para consertar Falsos Negativos
	"mapv": true, "filterv": true, "reduce-kv": true, "keys": true, "vals": true,
	"first": true, "last": true, "rest": true, "next": true,
	"assoc-in": true, "update-in": true, "get": true, "get-in": true, "concat": true, "reverse": true,
	"upper-case": true, "lower-case": true, "trim": true, "replace": true, "split": true,
	"count": true, "join": true, "str/join": true, "capitalize": true, "str/capitalize": true, "boolean": true,
}

func isThreadingCandidate(funcName string) bool {
	if threadingCandidateFunctions[funcName] {
		return true
	}
	parts := strings.Split(funcName, "/")
	if len(parts) == 2 {
		shortName := parts[1]
		return threadingCandidateFunctions[shortName] || threadingCandidateFunctions[funcName]
	}
	return false
}

func countNestedCalls(node *reader.RichNode, depth int) int {
	if node == nil || depth > 10 {
		return 0
	}
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return 0
	}
	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return 0
	}
	
	funcName := funcNode.Value
	switch funcName {
	case "fn", "let", "loop", "if", "when", "cond", "case", "def", "defn", "defn-", "try", "catch", "for", "doseq":
		return 0
	}

	count := 0
	if isThreadingCandidate(funcName) {
		count = 1
	}

	maxChildDepth := 0
	for i := 1; i < len(node.Children); i++ {
		child := node.Children[i]
		// Only recurse into direct function calls
		if child.Type == reader.NodeList {
			d := countNestedCalls(child, depth+1)
			if d > maxChildDepth {
				maxChildDepth = d
			}
		}
	}
	
	if count > 0 {
	    return count + maxChildDepth
	}
	return maxChildDepth
}

func nodeContainsSymbol(node *reader.RichNode, symbol string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSymbol && node.Value == symbol {
		return true
	}
	for _, child := range node.Children {
		if nodeContainsSymbol(child, symbol) {
			return true
		}
	}
	return false
}

func countLetChaining(node *reader.RichNode) int {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return 0
	}
	if node.Children[0].Type != reader.NodeSymbol || node.Children[0].Value != "let" {
		return 0
	}
	
	bindings := node.Children[1]
	if bindings.Type != reader.NodeVector {
		return 0
	}
	
	maxChain := 0
	currentChain := 0
	count := len(bindings.Children)
	
	for i := 2; i+1 < count; i += 2 {
		prevVarNode := bindings.Children[i-2]
		if prevVarNode == nil || prevVarNode.Type != reader.NodeSymbol {
			currentChain = 0
			continue
		}
		prevVarName := prevVarNode.Value
		
		currExpr := bindings.Children[i+1]
		if nodeContainsSymbol(currExpr, prevVarName) {
			currentChain++
			if currentChain > maxChain {
				maxChain = currentChain
			}
		} else {
			currentChain = 0
		}
	}
	return maxChain
}

func (r *ThreadIgnoranceRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	
	// Via 1: Detecta Data-Flow Chaining no bloco Let
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "let" {
		chainLinks := countLetChaining(node)
		if chainLinks >= r.MinLetChain {
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Data flow chaining detected in let bindings (%d sequential steps). Consider using threading macros (-> or ->>) instead of temporary variables for better readability.", chainLinks+1),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	// Via 2: Detecta Aninhamento Profundo Sintático (Nested Calls)
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		funcNode := node.Children[0]
		if funcNode.Type == reader.NodeSymbol {
			funcName := funcNode.Value
			// Aborta se o desenvolvedor já estiver usando uma macro de threading
			if funcName == "->" || funcName == "->>" || funcName == "some->" || funcName == "as->" || funcName == "cond->" || funcName == "cond->>" {
				return nil
			}
			
			if isThreadingCandidate(funcName) {
				depth := countNestedCalls(node, 0)
				if depth >= r.MinNestingDepth {
					return &rules.Finding{
						RuleID:   r.ID,
						Message:  fmt.Sprintf("Nested function calls detected (depth %d). Threading macros (-> or ->>) improve readability by eliminating nested parentheses and making data flow more explicit.", depth),
						Filepath: filepath,
						Location: node.Location,
						Severity: r.Severity,
					}
				}
			}
		}
	}

	return nil
}

func init() {
	defaultRule := &ThreadIgnoranceRule{
		Rule: rules.Rule{
			ID:          "thread-ignorance",
			Name:        "Thread Ignorance",
			Description: "Detects nested function calls or sequential let bindings that would benefit from threading macros (-> or ->>).",
			Severity:    rules.SeverityHint,
		},
		MinNestingDepth: 3, // Elevado de 2 para 3 para reduzir Falsos Positivos em aninhamentos básicos (Ex: (into [] (map ...)))
		MinLetChain:     2, // Exige no mínimo 3 variáveis (2 links/passagens de bastão) para sugerir threading
	}

	rules.RegisterRule(defaultRule)
}