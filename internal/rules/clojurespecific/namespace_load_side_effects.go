package clojurespecific

import (
	"fmt"
	"strings"
	"sync"

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
	// fileNsCount rastreia quantos (ns ...) foram encontrados por arquivo
	fileNsCount map[string]int
	// filePrecedingDefs rastreia se arquivo tem defs antes do require candidato
	filePrecedingDefs map[string]bool
	mu                sync.Mutex
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

	// Rastreia estado de arquivo: conta ns e defs
	r.mu.Lock()
	if r.fileNsCount == nil {
		r.fileNsCount = make(map[string]int)
		r.filePrecedingDefs = make(map[string]bool)
	}
	if symbol == "ns" {
		r.fileNsCount[filepath]++
		r.mu.Unlock()
		return nil
	}
	if symbol == "defn" || symbol == "defn-" || symbol == "def" || symbol == "defonce" ||
		symbol == "defmacro" || symbol == "defmulti" || symbol == "defprotocol" ||
		symbol == "defrecord" || symbol == "deftype" {
		r.filePrecedingDefs[filepath] = true
		r.mu.Unlock()
		return nil
	}
	nsCount := r.fileNsCount[filepath]
	hasPrecedingDefs := r.filePrecedingDefs[filepath]
	r.mu.Unlock()

	isLoadTime := isLoadTimeSideEffectSymbol(symbol)
	isLazyLoad := isLazyLoadSymbol(symbol)

	if !isLoadTime && !isLazyLoad {
		return nil
	}

	isInsideNs := false
	isInsideDefn := false
	isInsideComment := false

	if enclosing, ok := context["enclosingForms"].([]string); ok {
		for _, f := range enclosing {
			if f == "ns" {
				isInsideNs = true
			}
			if f == "defn" || f == "defn-" || f == "fn" || f == "letfn" {
				isInsideDefn = true
			}
			if f == "comment" {
				isInsideComment = true
			}
		}
	}

	// Dentro de (ns ...) é a forma correta
	if isInsideNs {
		return nil
	}

	// Dentro de (comment ...) é padrão REPL
	if isInsideComment {
		return nil
	}

	// requiring-resolve dentro de função é lazy loading proposital — aceitável
	if isLazyLoad && isInsideDefn {
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

	// Smell 2: require/use/import no top-level mas após outras definições no arquivo.
	// Situação problemática: o arquivo tem um segundo (ns ...) ou definições antes do require.
	// require imediatamente após o primeiro e único (ns ...) é idiomático — não reporta.
	if nsCount > 1 || hasPrecedingDefs {
		return &rules.Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf(
				"Namespace load side effect: '%s' appears after definitions in the file. "+
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
