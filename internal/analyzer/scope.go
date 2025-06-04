package analyzer

import (
	"strings"
)

var coreSymbols = map[string]SymbolType{

	"+": TypeCoreFunction, "-": TypeCoreFunction, "*": TypeCoreFunction, "/": TypeCoreFunction,
	"=": TypeCoreFunction, "<": TypeCoreFunction, ">": TypeCoreFunction, "<=": TypeCoreFunction, ">=": TypeCoreFunction,

	"map": TypeCoreFunction, "filter": TypeCoreFunction, "reduce": TypeCoreFunction,
	"vector": TypeCoreFunction, "list": TypeCoreFunction, "hash-map": TypeCoreFunction, "set": TypeCoreFunction,

	"str": TypeCoreFunction, "println": TypeCoreFunction, "get": TypeCoreFunction, "assoc": TypeCoreFunction,
	"dissoc": TypeCoreFunction, "get-in": TypeCoreFunction, "inc": TypeCoreFunction, "dec": TypeCoreFunction,

	"range": TypeCoreFunction, "count": TypeCoreFunction, "first": TypeCoreFunction, "rest": TypeCoreFunction,
	"concat": TypeCoreFunction, "into": TypeCoreFunction, "conj": TypeCoreFunction,

	"let": TypeCoreSpecialForm, "fn": TypeCoreSpecialForm, "defn": TypeCoreSpecialForm,
	"def": TypeCoreSpecialForm, "if": TypeCoreSpecialForm, "do": TypeCoreSpecialForm,
	"when": TypeCoreSpecialForm, "cond": TypeCoreSpecialForm, "loop": TypeCoreSpecialForm,
	"quote": TypeCoreSpecialForm, ".": TypeCoreSpecialForm, "new": TypeCoreSpecialForm,
	"try": TypeCoreSpecialForm, "catch": TypeCoreSpecialForm, "finally": TypeCoreSpecialForm,

	"true":  TypeVariable,
	"false": TypeVariable,
	"nil":   TypeVariable,

	"swap!":       TypeCoreFunction,
	"reset!":      TypeCoreFunction,
	"set!":        TypeCoreFunction,
	"alter-meta!": TypeCoreFunction,
	"reset-meta!": TypeCoreFunction,
}

func (s *Scope) Lookup(name string) (*SymbolInfo, bool) {

	current := s
	for current != nil {
		if info, found := current.symbols[name]; found {

			return info, true
		}
		current = current.parent
	}

	globalScope := s
	for globalScope != nil && globalScope.parent != nil {
		globalScope = globalScope.parent
	}

	if globalScope != nil {

		if !strings.Contains(name, "/") {
			if refInfo, found := globalScope.referredSymbols[name]; found {

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

		if strings.Contains(name, "/") {
			parts := strings.SplitN(name, "/", 2)
			if len(parts) == 2 {
				aliasPart := parts[0]
				if aliasInfo, found := globalScope.aliases[aliasPart]; found {

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

	if !strings.Contains(name, "/") {
		if coreType, found := coreSymbols[name]; found {

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

	if strings.Contains(name, ".") {

		synthInfo := &SymbolInfo{
			Name:            name,
			Definition:      nil,
			Type:            TypeJava,
			OriginNamespace: "",
			IsUsed:          false,
		}
		return synthInfo, true
	}

	return nil, false
}
