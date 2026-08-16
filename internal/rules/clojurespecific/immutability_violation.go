package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type immutabilityViolationRule struct{ rules.Rule }

func (r *immutabilityViolationRule) Meta() rules.Rule { return r.Rule }

func isLocalRuntimeScope(context map[string]interface{}) bool {
	for _, key := range []string{"isInsideFunction", "isInsideLet", "isInsideLoop", "isInsideBinding"} {
		if inside, _ := context[key].(bool); inside {
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

func (r *immutabilityViolationRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 ||
		strings.HasSuffix(filepath, "user.clj") || strings.Contains(filepath, "/dev/") {
		return nil
	}

	if rules.CallResolvesTo(node, "clojure.core/def", "clojure.core/defonce") &&
		isLocalRuntimeScope(context) && !isGeneratedOrMacroCode(context) {
		name := node.Children[0].Value
		return &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Found `%s` inside a local scope. This mutates global state and should be avoided.", name),
			Filepath: filepath, Location: node.Location, Severity: r.Severity,
		}
	}

	if rules.CallResolvesTo(node, "clojure.core/ref-set") {
		insideDosync, _ := context["isInsideDosync"].(bool)
		if !insideDosync && !isGeneratedOrMacroCode(context) {
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  "Found `ref-set` outside of `dosync`. Use `dosync` to ensure transactional safety with refs.",
				Filepath: filepath, Location: node.Location, Severity: r.Severity,
			}
		}
	}

	return nil
}

func init() {
	rules.RegisterRule(&immutabilityViolationRule{Rule: rules.Rule{
		ID: "immutability-violation", Name: "Immutability Violation",
		Description: "Detects high-confidence global state definitions in local runtime scopes and invalid ref-set use outside dosync.",
		Severity:    rules.SeverityWarning,
	}})
}
