// Package rules implementa regras de análise para código Clojure
package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// DefaultLazyContextFunctions define funções que criam contextos lazy por padrão
// Essas funções retornam sequências lazy que podem não executar imediatamente
var DefaultLazyContextFunctions = map[string]bool{
	"map":        true, // Aplica função a cada elemento (lazy)
	"filter":     true, // Filtra elementos baseado em predicado (lazy)
	"remove":     true, // Remove elementos baseado em predicado (lazy)
	"for":        true, // List comprehension (lazy)
	"lazy-seq":   true, // Cria sequência lazy explicitamente
	"mapcat":     true, // Map + concatenação (lazy)
	"lazy-cat":   true, // Concatenação lazy
	"keep":       true, // Map que remove nils (lazy)
	"distinct":   true, // Remove duplicatas (lazy)
	"interpose":  true, // Insere elemento entre itens (lazy)
	"iterate":    true, // Gera sequência infinita (lazy)
	"repeat":     true, // Repete valor infinitamente (lazy)
	"repeatedly": true, // Chama função repetidamente (lazy)
	"cycle":      true, // Cicla através de coleção (lazy)
}

// EagerConsumerFunctions define funções que consomem sequências eagerly
// Quando uma operação lazy é consumida por essas funções, os side effects são executados
var EagerConsumerFunctions = map[string]bool{
	"into":      true, // Coleta elementos em coleção
	"run!":      true, // Força execução para side effects
	"doseq":     true, // Loop com side effects
	"reduce":    true, // Reduz coleção a valor único
	"transduce": true, // Transformação + redução
	"mapv":      true, // Map que retorna vetor (eager)
	"dorun":     true, // Força execução sem coletar resultados
	"sequence":  true, // Converte para sequência realizando transformações
}

// DefaultSideEffectFunctions define funções que causam side effects
// Essas funções modificam estado ou produzem output, problemáticas em contextos lazy
var DefaultSideEffectFunctions = map[string]bool{
	"println":        true, // Output para console
	"print":          true, // Output para console sem newline
	"printf":         true, // Output formatado
	"prn":            true, // Print para leitura
	"pr":             true, // Print para leitura sem newline
	"swap!":          true, // Modifica atom
	"reset!":         true, // Reseta atom
	"add-watch":      true, // Adiciona watcher
	"remove-watch":   true, // Remove watcher
	"send":           true, // Envia para agent
	"send-off":       true, // Envia para agent (thread pool)
	"alter-var-root": true, // Modifica var global
	"spit":           true, // Escreve arquivo
	"aset":           true, // Modifica array
}

// LazySideEffectsRule detecta side effects em operações lazy
// Configura quais funções são consideradas lazy contexts e side effects
type LazySideEffectsRule struct {
	LazyContextFuncs map[string]bool `config:"lazy_context_funcs"` // Funções que criam contextos lazy
	SideEffectFuncs  map[string]bool `config:"side_effect_funcs"`  // Funções que causam side effects
}

func (r *LazySideEffectsRule) Meta() Rule {
	return Rule{
		ID:          "lazy-side-effects",
		Name:        "Lazy Side Effects",
		Description: "Detects potential side effects (like printing or state mutation) inside lazy sequence operations (map, filter, etc.). Side effects might not execute when expected. This rule ignores cases where the lazy operation is consumed by an eager function (e.g., 'into', 'run!', 'doseq').",
		Severity:    SeverityWarning,
	}
}

// maxRecursionDepth limita a profundidade de análise para evitar loops infinitos
const maxRecursionDepth = 50

// containsSideEffect verifica recursivamente se um nó contém side effects
// Usa mapa de visitados para evitar análise circular e limita profundidade
func containsSideEffect(node *reader.RichNode, visited map[*reader.RichNode]bool, sideEffects map[string]bool, currentDepth int, maxDepth int) bool {
	// Proteção contra recursão excessiva
	if currentDepth > maxDepth {
		return false
	}
	if node == nil || visited[node] {
		return false
	}
	visited[node] = true

	// Analisa chamadas de função
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		funcNode := node.Children[0]
		if funcNode.Type == reader.NodeSymbol {
			// Verifica se é um side effect direto
			if isDirectSideEffect, _ := sideEffects[funcNode.Value]; isDirectSideEffect {
				return true
			}

			// Analisa definição da função se disponível
			if funcNode.ResolvedDefinition != nil {
				definitionNode := funcNode.ResolvedDefinition
				var bodyNodesToAnalyze []*reader.RichNode

				// Extrai corpo de funções definidas com defn
				if definitionNode.Type == reader.NodeList && len(definitionNode.Children) > 0 {
					defSymbol := definitionNode.Children[0]
					if defSymbol.Type == reader.NodeSymbol && (defSymbol.Value == "defn" || defSymbol.Value == "defn-") {
						bodyNodesToAnalyze = extractDefnBodyNodes(definitionNode)
					}
				}

				// Analisa cada nó do corpo da função
				for _, bodyNode := range bodyNodesToAnalyze {
					// Novo mapa de visitados para cada análise de corpo
					newVisited := make(map[*reader.RichNode]bool)
					if containsSideEffect(bodyNode, newVisited, sideEffects, currentDepth+1, maxDepth) {
						return true
					}
				}
			}
		}
	}

	// Analisa recursivamente todos os filhos
	for _, child := range node.Children {
		if containsSideEffect(child, visited, sideEffects, currentDepth+1, maxDepth) {
			return true
		}
	}

	return false
}

func (r *LazySideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Ignora se já estamos em contexto eager (side effects serão executados)
	isInEagerCtx, _ := context["isInEagerContext"].(bool)

	if isInEagerCtx {
		return nil
	}

	// Verifica se é uma chamada de função
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return nil
	}
	lazyFuncName := funcNode.Value

	// Verifica se é uma função que cria contexto lazy
	if _, isLazyContext := r.LazyContextFuncs[lazyFuncName]; !isLazyContext {
		return nil
	}

	var funcArgNode *reader.RichNode
	if len(node.Children) >= 2 {
		// Tratamento especial para 'for' e 'lazy-seq' que têm sintaxe diferente
		if lazyFuncName == "for" && len(node.Children) > 2 && node.Children[1].Type == reader.NodeVector {
			// Para 'for', analisa as expressões do corpo (após o vetor de bindings)
			for i := 2; i < len(node.Children); i++ {
				bodyExpr := node.Children[i]
				visited := make(map[*reader.RichNode]bool)
				if containsSideEffect(bodyExpr, visited, r.SideEffectFuncs, 0, maxRecursionDepth) {
					return r.createFinding(node, lazyFuncName, "body expression", filepath, isInEagerCtx)
				}
			}
			return nil
		} else if lazyFuncName == "lazy-seq" {
			// Para 'lazy-seq', analisa todas as expressões do corpo
			for i := 1; i < len(node.Children); i++ {
				bodyExpr := node.Children[i]
				visited := make(map[*reader.RichNode]bool)
				if containsSideEffect(bodyExpr, visited, r.SideEffectFuncs, 0, maxRecursionDepth) {
					return r.createFinding(node, lazyFuncName, "body expression", filepath, isInEagerCtx)
				}
			}
			return nil
		} else {
			// Para outras funções lazy, o primeiro argumento é a função
			funcArgNode = node.Children[1]
		}
	}

	if funcArgNode == nil {
		return nil
	}

	var bodyToAnalyze *reader.RichNode
	funcNameStr := ""

	// Determina o tipo de função e o que analisar
	if funcArgNode.Type == reader.NodeFnLiteral {
		// Function literal: #(...)
		bodyToAnalyze = funcArgNode
		funcNameStr = "function literal (#(...))"
	} else if funcArgNode.Type == reader.NodeList && len(funcArgNode.Children) > 0 && funcArgNode.Children[0].Type == reader.NodeSymbol && funcArgNode.Children[0].Value == "fn" {
		// Function literal: (fn ...)
		bodyToAnalyze = funcArgNode
		funcNameStr = "function literal (fn ...)"
	} else if funcArgNode.Type == reader.NodeSymbol {
		// Símbolo de função - precisa resolver definição
		funcNameStr = fmt.Sprintf("symbol '%s'", funcArgNode.Value)
		if funcArgNode.ResolvedDefinition != nil {
			bodyToAnalyze = funcArgNode.ResolvedDefinition
		} else {
			// Se não conseguiu resolver, verifica se é side effect direto
			if _, isDirectSideEffect := r.SideEffectFuncs[funcArgNode.Value]; isDirectSideEffect {
				return r.createFinding(node, lazyFuncName, funcNameStr, filepath, isInEagerCtx)
			}
			return nil
		}
	} else {
		return nil
	}

	if bodyToAnalyze != nil {
		visited := make(map[*reader.RichNode]bool)
		if containsSideEffect(bodyToAnalyze, visited, r.SideEffectFuncs, 0, maxRecursionDepth) {
			return r.createFinding(node, lazyFuncName, funcNameStr, filepath, isInEagerCtx)
		}
	}

	return nil
}

func (r *LazySideEffectsRule) createFinding(node *reader.RichNode, lazyFuncName, funcSource string, filepath string, isInEagerCtx bool) *Finding {
	meta := r.Meta()

	messageSuffix := " Execution might be delayed or unexpected."
	if !isInEagerCtx {
		messageSuffix += " (Note: If this lazy call is ultimately consumed by an eager function like 'into' or 'run!', this warning might be a false positive)."
	}
	return &Finding{
		RuleID:   meta.ID,
		Message:  fmt.Sprintf("Function/symbol %s passed to lazy function '%s' may contain side effects.%s", funcSource, lazyFuncName, messageSuffix),
		Filepath: filepath,
		Location: node.Location,
		Severity: meta.Severity,
	}
}

func extractDefnBodyNodes(defnNode *reader.RichNode) []*reader.RichNode {
	bodyNodes := []*reader.RichNode{}
	if defnNode == nil || defnNode.Type != reader.NodeList || len(defnNode.Children) < 3 {
		return bodyNodes
	}

	currentIdx := 2

	if len(defnNode.Children) > currentIdx && defnNode.Children[currentIdx].Type == reader.NodeString {
		currentIdx++
	}

	if len(defnNode.Children) > currentIdx && defnNode.Children[currentIdx].Type == reader.NodeMap {
		currentIdx++
	}

	if len(defnNode.Children) <= currentIdx {
		return bodyNodes
	}

	if defnNode.Children[currentIdx].Type == reader.NodeList {
		multiArityList := defnNode.Children[currentIdx]
		for _, arityForm := range multiArityList.Children {

			if arityForm.Type == reader.NodeList && len(arityForm.Children) >= 2 && arityForm.Children[0].Type == reader.NodeVector {

				bodyNodes = append(bodyNodes, arityForm.Children[1:]...)
			}
		}
	} else if defnNode.Children[currentIdx].Type == reader.NodeVector {

		if len(defnNode.Children) > currentIdx+1 {
			bodyNodes = append(bodyNodes, defnNode.Children[currentIdx+1:]...)
		}
	}

	return bodyNodes
}

func init() {
	defaultRule := &LazySideEffectsRule{
		LazyContextFuncs: DefaultLazyContextFunctions,
		SideEffectFuncs:  DefaultSideEffectFunctions,
	}
	RegisterRule(defaultRule)
}
