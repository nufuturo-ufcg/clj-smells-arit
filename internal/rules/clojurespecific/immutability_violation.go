package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type ImmutabilityViolationRule struct {
	rules.Rule
}

func (r *ImmutabilityViolationRule) Meta() rules.Rule {
	return r.Rule
}

func isLocalRuntimeScope(context map[string]interface{}) bool {
	insideFunction, _ := context["isInsideFunction"].(bool)
	return insideFunction
}

func isKnownCoreMutation(node *reader.RichNode, names ...string) bool {
	if rules.CallResolvesTo(node, names...) {
		return true
	}
	if node == nil || len(node.Children) == 0 || node.Children[0] == nil {
		return false
	}
	head := node.Children[0]
	if head.Type != reader.NodeSymbol || strings.Contains(head.Value, "/") {
		return false
	}
	if head.Resolution != nil && head.Resolution.Kind != reader.ResolutionUnresolved {
		return false
	}
	for _, name := range names {
		if strings.TrimPrefix(name, "clojure.core/") == head.Value {
			return true
		}
	}
	return false
}

func isGeneratedOrMacroCode(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil {
			continue
		}
		switch ancestor.Type {
		case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeVarQuote, reader.NodeReaderDiscard:
			return true
		}
		if ancestor.Type == reader.NodeList && len(ancestor.Children) > 0 &&
			ancestor.Children[0].Type == reader.NodeSymbol && ancestor.Children[0].Value == "defmacro" {
			return true
		}
	}
	return false
}

// case constants are data, even when a grouped constant is represented by a
// list whose first symbol happens to be `def` or `defonce`.
func isCaseConstantPosition(context map[string]interface{}) bool {
	value, _ := context["isInCaseConstantPosition"].(bool)
	return value
}

func (r *ImmutabilityViolationRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 ||
		strings.HasSuffix(filepath, "user.clj") || strings.Contains(filepath, "/dev/") {
		return nil
	}

	if isGeneratedOrMacroCode(context) {
		return nil
	}
	if isCaseConstantPosition(context) {
		return nil
	}

	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return nil
	}

	if head.Resolution != nil && head.Resolution.Kind == reader.ResolutionLocal {
		return nil
	}

	symValue := head.Value
	unqualifiedSym := symValue
	if slash := strings.Index(symValue, "/"); slash >= 0 {
		unqualifiedSym = symValue[slash+1:]
	}

	if isKnownCoreMutation(node, "clojure.core/def", "clojure.core/defonce") &&
		isLocalRuntimeScope(context) {
		return &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Found `%s` inside a local scope. This mutates global state and should be avoided.", unqualifiedSym),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	if isKnownCoreMutation(node, "clojure.core/ref-set") {
		insideDosync, _ := context["isInsideDosync"].(bool)
		if !insideDosync {
			insideFunction, _ := context["isInsideFunction"].(bool)
			severity := r.Severity
			tags := []string(nil)
			if insideFunction {
				severity = rules.SeverityHint
				tags = []string{"low-confidence", "external-transaction-unknown"}
			}
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  "Found `ref-set` outside of `dosync`. Use `dosync` to ensure transactional safety with refs.",
				Filepath: filepath,
				Location: head.Location,
				Severity: severity,
				Tags:     tags,
			}
		}
	}

	return nil
}

func init() {
	rules.RegisterRule(&ImmutabilityViolationRule{Rule: rules.Rule{
		ID:          "immutability-violation",
		Name:        "Immutability Violation",
		Description: "Detects direct state mutation and violations of functional purity. Follows Clojure Style Guide recommendations for proper use of refs, atoms, agents, and avoiding global state mutation in local scopes.",
		Severity:    rules.SeverityWarning,
	}})
}
