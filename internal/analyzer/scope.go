package analyzer

import (
	"strings"
)

var coreSymbols = map[string]SymbolType{

	"+": TypeCoreFunction, "-": TypeCoreFunction, "*": TypeCoreFunction, "/": TypeCoreFunction,
	"=": TypeCoreFunction, "<": TypeCoreFunction, ">": TypeCoreFunction, "<=": TypeCoreFunction, ">=": TypeCoreFunction,

	"map": TypeCoreFunction, "mapv": TypeCoreFunction, "filter": TypeCoreFunction, "filterv": TypeCoreFunction, "reduce": TypeCoreFunction,
	"vec": TypeCoreFunction, "keep": TypeCoreFunction, "remove": TypeCoreFunction, "map-indexed": TypeCoreFunction,
	"keep-indexed": TypeCoreFunction, "take": TypeCoreFunction, "drop": TypeCoreFunction,
	"take-while": TypeCoreFunction, "drop-while": TypeCoreFunction, "distinct": TypeCoreFunction, "dedupe": TypeCoreFunction,
	"vector": TypeCoreFunction, "list": TypeCoreFunction, "hash-map": TypeCoreFunction, "set": TypeCoreFunction,

	"str": TypeCoreFunction, "println": TypeCoreFunction, "get": TypeCoreFunction, "assoc": TypeCoreFunction,
	"dissoc": TypeCoreFunction, "get-in": TypeCoreFunction, "inc": TypeCoreFunction, "dec": TypeCoreFunction,

	"range": TypeCoreFunction, "count": TypeCoreFunction, "first": TypeCoreFunction, "rest": TypeCoreFunction,
	"concat": TypeCoreFunction, "into": TypeCoreFunction, "conj": TypeCoreFunction,
	"int": TypeCoreFunction, "long": TypeCoreFunction, "unchecked-int": TypeCoreFunction,
	"unchecked-long": TypeCoreFunction, "compare": TypeCoreFunction,

	"let": TypeCoreSpecialForm, "fn": TypeCoreSpecialForm, "defn": TypeCoreSpecialForm,
	"def": TypeCoreSpecialForm, "defonce": TypeCoreSpecialForm, "if": TypeCoreSpecialForm, "do": TypeCoreSpecialForm,
	"when": TypeCoreSpecialForm, "when-not": TypeCoreSpecialForm,
	"if-not": TypeCoreSpecialForm, "while": TypeCoreSpecialForm,
	"and": TypeCoreSpecialForm, "or": TypeCoreSpecialForm, "cond": TypeCoreSpecialForm, "loop": TypeCoreSpecialForm,
	"quote": TypeCoreSpecialForm, ".": TypeCoreSpecialForm, "new": TypeCoreSpecialForm,
	"try": TypeCoreSpecialForm, "catch": TypeCoreSpecialForm, "finally": TypeCoreSpecialForm,

	"true":  TypeVariable,
	"false": TypeVariable,
	"nil":   TypeVariable,

	"swap!":       TypeCoreFunction,
	"reset!":      TypeCoreFunction,
	"atom":        TypeCoreFunction,
	"agent":       TypeCoreFunction,
	"ref":         TypeCoreFunction,
	"volatile!":   TypeCoreFunction,
	"send":        TypeCoreFunction,
	"send-off":    TypeCoreFunction,
	"alter":       TypeCoreFunction,
	"commute":     TypeCoreFunction,
	"ref-set":     TypeCoreFunction,
	"vreset!":     TypeCoreFunction,
	"vswap!":      TypeCoreFunction,
	"set!":        TypeCoreFunction,
	"alter-meta!": TypeCoreFunction,
	"reset-meta!": TypeCoreFunction,
	"empty?":      TypeCoreFunction,
	"zero?":       TypeCoreFunction,
	"pos?":        TypeCoreFunction,
	"neg?":        TypeCoreFunction,
	"not":         TypeCoreFunction,
	"not=":        TypeCoreFunction,
	"==":          TypeCoreFunction,
	"mod":         TypeCoreFunction,
	"rem":         TypeCoreFunction,
	"boolean":     TypeCoreFunction,
	"defprotocol": TypeCoreFunction,
	"future":      TypeCoreFunction,
	"delay":       TypeCoreFunction,
	"lazy-seq":    TypeCoreFunction,
	"letfn":       TypeCoreSpecialForm,
	"dosync":      TypeCoreSpecialForm,

	"for":           TypeCoreSpecialForm,
	"mapcat":        TypeCoreFunction,
	"flatten":       TypeCoreFunction,
	"interleave":    TypeCoreFunction,
	"interpose":     TypeCoreFunction,
	"partition":     TypeCoreFunction,
	"partition-all": TypeCoreFunction,
	"partition-by":  TypeCoreFunction,
	"take-nth":      TypeCoreFunction,
	"cycle":         TypeCoreFunction,
	"repeat":        TypeCoreFunction,
	"iterate":       TypeCoreFunction,
	"seq":           TypeCoreFunction,
	"seq?":          TypeCoreFunction,
	"nil?":          TypeCoreFunction,
	"some?":         TypeCoreFunction,
	"true?":         TypeCoreFunction,
	"false?":        TypeCoreFunction,
	"boolean?":      TypeCoreFunction,
	"string?":       TypeCoreFunction,
	"number?":       TypeCoreFunction,
	"deref":         TypeCoreFunction,
	"realized?":     TypeCoreFunction,
	"keys":          TypeCoreFunction,
	"vals":          TypeCoreFunction,
	"rseq":          TypeCoreFunction,
	"subvec":        TypeCoreFunction,
	"reverse":       TypeCoreFunction,
	"sort":          TypeCoreFunction,
	"sort-by":       TypeCoreFunction,

	"load":              TypeCoreFunction,
	"load-file":         TypeCoreFunction,
	"in-ns":             TypeCoreFunction,
	"require":           TypeCoreFunction,
	"use":               TypeCoreFunction,
	"import":            TypeCoreFunction,
	"requiring-resolve": TypeCoreFunction,
	"slurp":             TypeCoreFunction,
	"spit":              TypeCoreFunction,
	"doall":             TypeCoreFunction,
	"dorun":             TypeCoreFunction,
}

func (s *Scope) Lookup(name string) (*SymbolInfo, bool) {
	if s == nil || name == "" {
		return nil, false
	}

	current := s
	for current != nil {
		if current.symbols != nil {
			if info, found := current.symbols[name]; found && info != nil {
				return info, true
			}
		}
		current = current.parent
	}

	globalScope := s
	for globalScope != nil && globalScope.parent != nil {
		globalScope = globalScope.parent
	}

	if globalScope != nil {

		if !strings.Contains(name, "/") {
			if globalScope.referredSymbols != nil {
				if refInfo, found := globalScope.referredSymbols[name]; found && refInfo != nil {
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
		}

		if strings.Contains(name, "/") {
			parts := strings.SplitN(name, "/", 2)
			if len(parts) == 2 {
				aliasPart := parts[0]
				if globalScope.aliases != nil {
					if aliasInfo, found := globalScope.aliases[aliasPart]; found && aliasInfo != nil {
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

	return nil, false
}
