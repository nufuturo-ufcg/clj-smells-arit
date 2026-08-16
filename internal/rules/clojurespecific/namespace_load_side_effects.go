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
	lowerPath := strings.ToLower(filepath)
	if !strings.Contains(lowerPath, "internal/test/data") {
		if strings.HasSuffix(lowerPath, "project.clj") ||
			strings.HasSuffix(lowerPath, "deps.edn") ||
			strings.Contains(lowerPath, "/support/") ||
			strings.Contains(lowerPath, "/scripts/") ||
			strings.Contains(lowerPath, "/dev/") ||
			strings.Contains(lowerPath, "/build/") ||
			strings.Contains(lowerPath, "/repl/") ||
			strings.HasSuffix(lowerPath, "_test.clj") {
			return nil
		}
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
			if f == "defn" || f == "defn-" || f == "fn" || f == "letfn" {
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
		return &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Side effect: '%s' called inside a function body. Move namespace dependencies to the (ns ...) :require form.", symbol),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	// A direct top-level form is tolerated for scripts. Nested top-level
	// execution (def initializers, conditionals, try, let, etc.) is hidden
	// load-time behavior and is therefore reportable.
	if parent, ok := context["parent"].(*reader.RichNode); ok && parent != nil {
		return &rules.Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf(
				"Namespace load side effect: '%s' is nested in a load-time expression. "+
					"All namespace dependencies should be declared inside the (ns ...) macro at the top of the file.",
				symbol,
			),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
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
