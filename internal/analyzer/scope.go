// Package analyzer implementa a análise semântica de código Clojure
package analyzer

import (
	"strings"
)

// coreSymbols mapeia símbolos do core do Clojure para seus tipos
// Inclui funções básicas, formas especiais e constantes fundamentais
var coreSymbols = map[string]SymbolType{
	// Operadores aritméticos e de comparação
	"+": TypeCoreFunction, "-": TypeCoreFunction, "*": TypeCoreFunction, "/": TypeCoreFunction,
	"=": TypeCoreFunction, "<": TypeCoreFunction, ">": TypeCoreFunction, "<=": TypeCoreFunction, ">=": TypeCoreFunction,

	// Funções de manipulação de coleções
	"map": TypeCoreFunction, "filter": TypeCoreFunction, "reduce": TypeCoreFunction,
	"vector": TypeCoreFunction, "list": TypeCoreFunction, "hash-map": TypeCoreFunction, "set": TypeCoreFunction,

	// Funções de string e acesso a dados
	"str": TypeCoreFunction, "println": TypeCoreFunction, "get": TypeCoreFunction, "assoc": TypeCoreFunction,
	"dissoc": TypeCoreFunction, "get-in": TypeCoreFunction, "inc": TypeCoreFunction, "dec": TypeCoreFunction,

	// Funções de sequência e manipulação
	"range": TypeCoreFunction, "count": TypeCoreFunction, "first": TypeCoreFunction, "rest": TypeCoreFunction,
	"concat": TypeCoreFunction, "into": TypeCoreFunction, "conj": TypeCoreFunction,

	// Formas especiais fundamentais do Clojure
	"let": TypeCoreSpecialForm, "fn": TypeCoreSpecialForm, "defn": TypeCoreSpecialForm,
	"def": TypeCoreSpecialForm, "if": TypeCoreSpecialForm, "do": TypeCoreSpecialForm,
	"when": TypeCoreSpecialForm, "cond": TypeCoreSpecialForm, "loop": TypeCoreSpecialForm,
	"quote": TypeCoreSpecialForm, ".": TypeCoreSpecialForm, "new": TypeCoreSpecialForm,
	"try": TypeCoreSpecialForm, "catch": TypeCoreSpecialForm, "finally": TypeCoreSpecialForm,

	// Constantes básicas
	"true":  TypeVariable,
	"false": TypeVariable,
	"nil":   TypeVariable,

	// Funções de mutação (com ! por convenção)
	"swap!":       TypeCoreFunction,
	"reset!":      TypeCoreFunction,
	"set!":        TypeCoreFunction,
	"alter-meta!": TypeCoreFunction,
	"reset-meta!": TypeCoreFunction,
}

// Lookup busca um símbolo no escopo atual e nos escopos pai
// Implementa a resolução de símbolos seguindo as regras do Clojure
func (s *Scope) Lookup(name string) (*SymbolInfo, bool) {
	// Primeiro, busca nos escopos locais (de dentro para fora)
	current := s
	for current != nil {
		if info, found := current.symbols[name]; found {
			// Símbolo encontrado no escopo local
			return info, true
		}
		current = current.parent
	}

	// Se não encontrou localmente, busca no escopo global
	globalScope := s
	for globalScope != nil && globalScope.parent != nil {
		globalScope = globalScope.parent
	}

	if globalScope != nil {
		// Busca por símbolos referidos (via :refer)
		if !strings.Contains(name, "/") {
			if refInfo, found := globalScope.referredSymbols[name]; found {
				// Cria SymbolInfo sintético para símbolo referido
				synthInfo := &SymbolInfo{
					Name:            name,
					Definition:      refInfo.DefinitionNode,
					Type:            TypeReferred,
					OriginNamespace: refInfo.OriginalNamespace,
					IsUsed:          false,
				}
				return synthInfo, true
			}
		}

		// Busca por símbolos com alias (namespace/symbol)
		if strings.Contains(name, "/") {
			parts := strings.SplitN(name, "/", 2)
			if len(parts) == 2 {
				aliasPart := parts[0]
				if aliasInfo, found := globalScope.aliases[aliasPart]; found {
					// Cria SymbolInfo sintético para símbolo com alias
					synthInfo := &SymbolInfo{
						Name:            name,
						Definition:      aliasInfo.DefinitionNode,
						Type:            TypeAliased,
						OriginNamespace: aliasInfo.FullNamespace,
						IsUsed:          false,
					}
					return synthInfo, true
				}
			}
		}
	}

	// Busca nos símbolos do core do Clojure
	if !strings.Contains(name, "/") {
		if coreType, found := coreSymbols[name]; found {
			// Cria SymbolInfo sintético para símbolo do core
			synthInfo := &SymbolInfo{
				Name:            name,
				Definition:      nil,
				Type:            coreType,
				OriginNamespace: "clojure.core",
				IsUsed:          false,
			}
			return synthInfo, true
		}
	}

	// Verifica se é uma classe Java (contém ponto)
	if strings.Contains(name, ".") {
		// Assume que é uma classe ou método Java
		synthInfo := &SymbolInfo{
			Name:            name,
			Definition:      nil,
			Type:            TypeJava,
			OriginNamespace: "",
			IsUsed:          false,
		}
		return synthInfo, true
	}

	// Símbolo não encontrado
	return nil, false
}
