package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type ThreadIgnoranceRule struct {
	Rule
	MinNestingDepth int `json:"min_nesting_depth" yaml:"min_nesting_depth"`
	MaxArguments    int `json:"max_arguments" yaml:"max_arguments"`
}

func (r *ThreadIgnoranceRule) Meta() Rule {
	return r.Rule
}

var threadingCandidateFunctions = map[string]bool{

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

	"assoc":       true,
	"dissoc":      true,
	"update":      true,
	"merge":       true,
	"select-keys": true,

	"str/replace":    true,
	"str/trim":       true,
	"str/upper-case": true,
	"str/lower-case": true,
	"str/split":      true,

	"vec":  true,
	"set":  true,
	"seq":  true,
	"into": true,
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

	count := 0

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		if funcNode := node.Children[0]; funcNode.Type == reader.NodeSymbol {
			if isThreadingCandidate(funcNode.Value) {
				count = 1
			}
		}

		for i := 1; i < len(node.Children); i++ {
			count += countNestedCalls(node.Children[i], depth+1)
		}
	} else {

		for _, child := range node.Children {
			count += countNestedCalls(child, depth+1)
		}
	}

	return count
}

func hasNestedThreadingCandidates(node *reader.RichNode) (bool, int, string) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false, 0, ""
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return false, 0, ""
	}

	funcName := funcNode.Value

	if !isThreadingCandidate(funcName) {
		return false, 0, ""
	}

	totalCalls := countNestedCalls(node, 0)

	if totalCalls >= 2 {

		threadingType := "thread-first"
		if isCollectionFunction(funcName) {
			threadingType = "thread-last"
		}

		return true, totalCalls, threadingType
	}

	return false, totalCalls, ""
}

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

func generateThreadingExample(node *reader.RichNode, threadingType string) string {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return ""
	}

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

func (r *ThreadIgnoranceRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		if funcNode := node.Children[0]; funcNode.Type == reader.NodeSymbol {
			funcName := funcNode.Value
			if funcName == "->" || funcName == "->>" || funcName == "some->" || funcName == "as->" || funcName == "cond->" || funcName == "cond->>" {
				return nil
			}
		}
	}

	hasNested, callCount, threadingType := hasNestedThreadingCandidates(node)

	if !hasNested || callCount < r.MinNestingDepth {
		return nil
	}

	if node.Type == reader.NodeList && len(node.Children) > r.MaxArguments {
		return nil
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

func init() {
	defaultRule := &ThreadIgnoranceRule{
		Rule: Rule{
			ID:          "thread-ignorance",
			Name:        "Thread Ignorance",
			Description: "Detects nested function calls that would benefit from threading macros (-> or ->>). Threading macros improve readability by making data transformations more explicit and reducing nested parentheses. This rule suggests using -> for 'thread-first' patterns and ->> for 'thread-last' patterns.",
			Severity:    SeverityHint,
		},
		MinNestingDepth: 2,
		MaxArguments:    8,
	}

	RegisterRule(defaultRule)
}
