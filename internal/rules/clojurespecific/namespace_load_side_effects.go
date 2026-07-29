package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type NamespaceLoadSideEffectsRule struct {
	rules.Rule
}

func (r *NamespaceLoadSideEffectsRule) Meta() rules.Rule {
	return r.Rule
}

func isLoadTimeSideEffectSymbol(symbol string) bool {
	switch symbol {
	case "require", "use", "import", "load", "load-file":
		return true
	}
	return false
}

func isLazyLoadSymbol(symbol string) bool {
	return symbol == "requiring-resolve"
}

func (r *NamespaceLoadSideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	// 1. Context Awareness: Ignora ambientes onde imperativos soltos são o padrão da linguagem
	lowerPath := strings.ToLower(filepath)
	if strings.HasSuffix(lowerPath, "project.clj") || 
	   strings.HasSuffix(lowerPath, "deps.edn") || 
	   strings.Contains(lowerPath, "/support/") || 
	   strings.Contains(lowerPath, "/scripts/") ||
	   strings.Contains(lowerPath, "/test/") ||
	   strings.Contains(lowerPath, "/dev/") ||
	   strings.Contains(lowerPath, "/build/") ||
	   strings.Contains(lowerPath, "/repl/") {
		return nil
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		symbol := node.Children[0].Value

		isLoadTime := isLoadTimeSideEffectSymbol(symbol)
		isLazyLoad := isLazyLoadSymbol(symbol)

		// 2. Expansão de Símbolos: Captura tanto require clássico quanto requerimentos dinâmicos
		if isLoadTime || isLazyLoad {
			isInsideNs := false
			isInsideDefn := false

			if enclosing, ok := context["enclosingForms"].([]string); ok {
				for _, f := range enclosing {
					if f == "ns" {
						isInsideNs = true
					}
					// Se estiver dentro de uma função, consideramos escopo de execução (Runtime)
					if f == "defn" || f == "defn-" || f == "fn" {
						isInsideDefn = true
					}
				}
			}

			// 3. Inspeção de Escopo Semântico
			
			// Se o side-effect de dependência está protegido e envelopado pelo (ns ...), é a forma correta.
			if isInsideNs {
				return nil
			}

			// Se é um lazy load (requiring-resolve) executado dentro de uma função, não é Load-Time Side-Effect!
			// O desenvolvedor está fazendo Lazy Loading propositalmente, o que é uma excelente prática.
			if isLazyLoad && isInsideDefn {
				return nil
			}

			// Caso contrário, é um verdadeiro side-effect em Load-Time (seja imperativo solto ou dentro de um def top-level)
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Side effect: '%s' detected outside of ns macro. This bypasses the static dependency graph and pollutes the load-time environment.", symbol),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}
	return nil
}

func init() {
	defaultRule := &NamespaceLoadSideEffectsRule{
		Rule: rules.Rule{
			ID:          "namespace-load-side-effects",
			Name:        "Namespace Load Side Effects",
			Description: "Using require, use, or import outside a ns macro introduces hidden dependencies. requiring-resolve at top-level causes dynamic load side effects during compilation.",
			Severity:    rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}