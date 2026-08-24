package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

// NamespaceLoadSideEffectsRule detecta require/use/import em locais problemáticos:
// 1. Dentro do corpo de funções (defn, fn)
// 2. No top-level APÓS outras definições (segundo ns, defn, def, etc.)
//
// O padrão idiomático em Clojure é declarar todas as dependências no bloco (ns ...) :require.
// require top-level imediatamente após o (ns ...) é tolerado (estilo de scripts).
// O smell real é quando require aparece DEPOIS de definições no arquivo.
type NamespaceLoadSideEffectsRule struct {
	rules.Rule
}

func (r *NamespaceLoadSideEffectsRule) Meta() rules.Rule {
	return r.Rule
}

func isLoadTimeSideEffectCall(node *reader.RichNode) bool {
	return rules.CallResolvesTo(node,
		"clojure.core/require", "clojure.core/use", "clojure.core/import", "clojure.core/load-file")
}

func isLazyLoadCall(node *reader.RichNode) bool {
	return rules.CallResolvesTo(node, "clojure.core/requiring-resolve")
}

func (r *NamespaceLoadSideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	lowerPath := strings.ToLower(filepath)
	if strings.HasSuffix(lowerPath, "project.clj") || strings.HasSuffix(lowerPath, "deps.edn") {
		return nil
	}

	if node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0].Type != reader.NodeSymbol {
		return nil
	}

	symbol := node.Children[0].Value

	if symbol == "ns" {
		return nil
	}
	if !rules.ExecutesAtLoad(context) {
		return nil
	}

	isLoadTime := isLoadTimeSideEffectCall(node)
	isLazyLoad := isLazyLoadCall(node)

	if !isLoadTime && !isLazyLoad {
		return nil
	}

	isInsideNs := false
	isInsideDefn := false

	if enclosing, ok := context["enclosingForms"].([]string); ok {
		for _, f := range enclosing {
			if f == "ns" {
				isInsideNs = true
			}
			if f == "defn" || f == "defn-" || f == "fn" || f == "letfn" || f == "__fn-literal__" {
				isInsideDefn = true
			}
		}
	}

	// Dentro de (ns ...) é a forma correta
	if isInsideNs {
		return nil
	}

	// requiring-resolve dentro de função é lazy loading proposital — aceitável
	if isLazyLoad && isInsideDefn {
		return nil
	}

	// `(require ns-sym)` is a deliberate runtime plugin-loading pattern. Static
	// dependency rules cannot prove it should be eager, so prefer a false
	// negative to a false positive.
	if symbol == "require" && len(node.Children) == 2 && node.Children[1].Type == reader.NodeSymbol {
		return nil
	}
	// Smell 1: require/use/import DENTRO do corpo de uma função
	if isInsideDefn {
		severity := rules.ContextualSeverity(context, r.Severity)
		tags := rules.ContextualTags(context)
		if hasOptionalLoadGuard(context) {
			severity = rules.SeverityHint
			tags = append(tags, "conditional-load")
		}
		return &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Side effect: '%s' called inside a function body. Move namespace dependencies to the (ns ...) :require form.", symbol),
			Filepath: filepath,
			Location: node.Location,
			Severity: severity,
			Tags:     tags,
		}
	}

	// A direct top-level form is tolerated for scripts. Nested top-level
	// execution (def initializers, conditionals, try, let, etc.) is hidden
	// load-time behavior and is therefore reportable.
	if parent, ok := context["parent"].(*reader.RichNode); ok && parent != nil {
		severity := rules.ContextualSeverity(context, r.Severity)
		tags := rules.ContextualTags(context)
		if hasOptionalLoadGuard(context) {
			severity = rules.SeverityHint
			tags = append(tags, "conditional-load")
		}
		return &rules.Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf(
				"Namespace load side effect: '%s' is nested in a load-time expression. "+
					"All namespace dependencies should be declared inside the (ns ...) macro at the top of the file.",
				symbol,
			),
			Filepath: filepath,
			Location: node.Location,
			Severity: severity,
			Tags:     tags,
		}
	}

	return nil
}

func hasOptionalLoadGuard(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		head := strings.TrimPrefix(ancestor.Children[0].Value, "clojure.core/")
		switch head {
		case "if", "when", "when-not", "if-not":
			if len(ancestor.Children) > 1 && containsOptionalGuardCall(ancestor.Children[1]) {
				return true
			}
		case "try":
			for _, child := range ancestor.Children[1:] {
				if child != nil && child.Type == reader.NodeList && len(child.Children) > 0 && child.Children[0].Type == reader.NodeSymbol && child.Children[0].Value == "catch" {
					return true
				}
			}
		}
	}
	return false
}

func containsOptionalGuardCall(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		head := node.Children[0].Value
		if head == "resolve" || head == "clojure.core/resolve" ||
			head == "System/getProperty" || head == "System/getenv" ||
			head == "java.lang.System/getProperty" || head == "java.lang.System/getenv" {
			return true
		}
	}
	for _, child := range node.Children {
		if containsOptionalGuardCall(child) {
			return true
		}
	}
	return false
}

func init() {
	rules.RegisterRule(&NamespaceLoadSideEffectsRule{
		Rule: rules.Rule{
			ID:          "namespace-load-side-effects",
			Name:        "Namespace Load Side Effects",
			Description: "Using require, use, or import inside function bodies or after other top-level definitions introduces hidden dependencies. Declare all namespace dependencies in the (ns ...) :require form.",
			Severity:    rules.SeverityWarning,
		},
	})
}
