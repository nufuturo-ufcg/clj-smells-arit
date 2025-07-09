package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type InappropriateCollectionRule struct{}

func (r *InappropriateCollectionRule) Meta() Rule {
	return Rule{
		ID:          "inappropriate-collection",
		Name:        "Inappropriate Collection Usage",
		Description: "Detects inefficient or non-idiomatic usage of collection functions. This includes using 'last' on vectors (consider 'peek'), 'nth' on lists for random access, 'contains?' on lists/vectors for key lookups instead of sets/maps, and various other anti-patterns.",
		Severity:    SeverityHint,
	}
}

func getCollectionNodeType(node *reader.RichNode) reader.NodeType {
	if node == nil {
		return reader.NodeUnknown
	}

	// Direct literal detection
	if node.Type == reader.NodeVector || node.Type == reader.NodeList || node.Type == reader.NodeMap || node.Type == reader.NodeSet {
		return node.Type
	}

	// Quote detection - '(...) is a list literal
	if node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "quote" {
		if len(node.Children) > 1 {
			quotedNode := node.Children[1]
			if quotedNode.Type == reader.NodeList {
				return reader.NodeList
			}
			if quotedNode.Type == reader.NodeVector {
				return reader.NodeVector
			}
		}
	}

	if node.Type == reader.NodeSymbol {
		if node.ResolvedDefinition != nil {
			defNode := node.ResolvedDefinition

			if defNode.Type == reader.NodeVector || defNode.Type == reader.NodeMap || defNode.Type == reader.NodeSet {
				return defNode.Type
			}

			if defNode.Type == reader.NodeList && len(defNode.Children) > 0 {
				defTypeSymbolNode := defNode.Children[0]

				if defTypeSymbolNode.Type == reader.NodeSymbol {
					defFormName := defTypeSymbolNode.Value
					if (defFormName == "def" || defFormName == "defonce") && len(defNode.Children) >= 3 {
						valueNode := defNode.Children[len(defNode.Children)-1]

						if valueNode.Type == reader.NodeList && len(valueNode.Children) > 0 && valueNode.Children[0].Type == reader.NodeSymbol {
							constructorFuncNode := valueNode.Children[0]
							constructorFuncName := constructorFuncNode.Value

							switch constructorFuncName {
							case "vector", "vec":
								return reader.NodeVector
							case "list", "list*":
								return reader.NodeList
							case "hash-map", "array-map", "sorted-map", "sorted-map-by":
								return reader.NodeMap
							case "map":
								return reader.NodeList
							case "hash-set", "set", "sorted-set", "sorted-set-by":
								return reader.NodeSet
							case "into":
								if len(valueNode.Children) > 1 {
									targetCollectionNode := valueNode.Children[1]

									if targetCollectionNode.Type == reader.NodeMap {
										return reader.NodeMap
									}
									if targetCollectionNode.Type == reader.NodeSet {
										return reader.NodeSet
									}
									if targetCollectionNode.Type == reader.NodeVector {
										return reader.NodeVector
									}
									if targetCollectionNode.Type == reader.NodeList {
										return reader.NodeList
									}

									if targetCollectionNode.Type == reader.NodeSymbol {
										resolvedTargetType := getCollectionNodeType(targetCollectionNode)
										if resolvedTargetType != reader.NodeUnknown {
											return resolvedTargetType
										}
									}

									return reader.NodeList
								} else {
									return reader.NodeUnknown
								}
							default:
								return reader.NodeList
							}
						}

						if valueNode.Type == reader.NodeVector || valueNode.Type == reader.NodeMap || valueNode.Type == reader.NodeSet {
							return valueNode.Type
						}

						if valueNode.Type == reader.NodeList {
							return reader.NodeList
						}
					}
				}
			}
		}
	}

	return reader.NodeUnknown
}

// detectQuotedList detecta listas literais com quote syntax '(...)
func detectQuotedList(node *reader.RichNode) bool {
	// Para '(...), o parser pode criar diferentes estruturas
	if node.Type == reader.NodeList && len(node.Children) == 2 {
		if firstChild := node.Children[0]; firstChild.Type == reader.NodeSymbol && firstChild.Value == "quote" {
			if secondChild := node.Children[1]; secondChild.Type == reader.NodeList {
				return true
			}
		}
	}

	// Verificar se é uma quote form diretamente (syntax reader pode variar)
	if node.Type == reader.NodeQuote {
		if len(node.Children) > 0 && node.Children[0].Type == reader.NodeList {
			return true
		}
	}

	return false
}

func (r *InappropriateCollectionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return nil
	}

	funcName := funcNode.Value
	collectionNode := node.Children[1]
	collectionType := getCollectionNodeType(collectionNode)

	var message string
	report := false

	switch funcName {
	case "last":
		if collectionType == reader.NodeVector {
			message = "Using 'last' on a vector can be inefficient for large vectors. Consider 'peek' or rethink data structure if last-element access is frequent."
			report = true
		}
	case "nth":
		// Detectar lista literal '(...)
		if detectQuotedList(collectionNode) {
			message = "Using 'nth' for random access on a list is inefficient (linear time). Use a vector if random access is needed."
			report = true
		} else if collectionType == reader.NodeList {
			message = "Using 'nth' for random access on a list is inefficient (linear time). Use a vector if random access is needed."
			report = true
		} else if collectionNode.Type == reader.NodeQuote {
			// Detecção direta de quote node
			message = "Using 'nth' for random access on a list is inefficient (linear time). Use a vector if random access is needed."
			report = true
		}
	case "some":
		// Detectar padrão some #(= % value) collection
		if len(node.Children) >= 3 {
			predNode := node.Children[1]
			targetCollectionNode := node.Children[2]

			// Verificar se é fn literal #(= % value) e se collection é vector
			if predNode.Type == reader.NodeFnLiteral && targetCollectionNode.Type == reader.NodeVector {
				message = "Using 'some' for membership check on a vector is inefficient. Use a set with 'contains?' for O(1) membership testing."
				report = true
			}
		}
	case "first":
		// Detectar first + filter pattern
		if len(node.Children) >= 2 && collectionNode.Type == reader.NodeList && len(collectionNode.Children) >= 2 {
			if collectionNode.Children[0].Type == reader.NodeSymbol && collectionNode.Children[0].Value == "filter" {
				message = "Using '(first (filter ...))' is inefficient. Consider using 'some' which stops after the first match."
				report = true
			}
		}
	case "empty?":
		// Detectar empty? + filter pattern
		if len(node.Children) >= 2 && collectionNode.Type == reader.NodeList && len(collectionNode.Children) >= 2 {
			if collectionNode.Children[0].Type == reader.NodeSymbol && collectionNode.Children[0].Value == "filter" {
				message = "Using '(empty? (filter ...))' is inefficient. Consider using 'not-any?' for early termination."
				report = true
			}
		}
	case "count":
		// Detectar count + filter pattern
		if len(node.Children) >= 2 && collectionNode.Type == reader.NodeList && len(collectionNode.Children) >= 2 {
			if collectionNode.Children[0].Type == reader.NodeSymbol && collectionNode.Children[0].Value == "filter" {
				message = "Using '(count (filter ...))' processes entire collection. Consider 'transduce' with counting reducer for potentially better performance."
				report = true
			}
		}
	case "sequence":
		// Detectar sequence + mapcat pattern (performance issue)
		if len(node.Children) >= 3 {
			firstArg := node.Children[1]
			if firstArg.Type == reader.NodeList && len(firstArg.Children) >= 2 {
				if firstArg.Children[0].Type == reader.NodeSymbol && firstArg.Children[0].Value == "mapcat" {
					message = "Using '(sequence (mapcat ...))' can cause memory issues with large lazy sequences. Consider using 'mapcat' directly or 'transduce' for better performance."
					report = true
				}
			}
		}
	case "contains?":
		if collectionType == reader.NodeVector || collectionType == reader.NodeList {
			if len(node.Children) > 2 {
				keyNode := node.Children[2]
				if keyNode.Type == reader.NodeKeyword || keyNode.Type == reader.NodeString || keyNode.Type == reader.NodeSymbol {
					message = fmt.Sprintf("Using 'contains?' with a non-numeric key on a %s is inefficient. Use a set for presence checks or a map for key lookups.", collectionType)
					report = true
				}
			}
		}
	case "reduce":
		// Detectar reduce acumulando numa lista vazia '()
		if len(node.Children) >= 3 {
			accumInitNode := node.Children[2]
			if detectQuotedList(accumInitNode) || accumInitNode.Type == reader.NodeQuote {
				message = "Using 'reduce' with a list as accumulator can be inefficient. Consider using a vector for better performance."
				report = true
			}
		}
	case "map":
		// Detectar map aplicado em hash-map (não preserva ordem)
		if len(node.Children) >= 3 {
			targetCollectionNode := node.Children[len(node.Children)-1]
			if targetCollectionNode.Type == reader.NodeMap {
				message = "Using 'map' on a hash-map doesn't preserve order. Consider using a vector of pairs or sorted-map if order matters."
				report = true
			}
		}
		// Detectar map com identity (desnecessário)
		if len(node.Children) >= 2 {
			funcArg := node.Children[1]
			if funcArg.Type == reader.NodeSymbol && funcArg.Value == "identity" {
				message = "Using 'map' with 'identity' is unnecessary. Consider using 'seq' or removing the transformation entirely."
				report = true
			}
		}
	case "apply":
		// Detectar apply hash-map em vetor (conversão desnecessária)
		if len(node.Children) >= 3 {
			funcAppliedNode := node.Children[1]
			dataNode := node.Children[2]
			if funcAppliedNode.Type == reader.NodeSymbol && funcAppliedNode.Value == "hash-map" {
				// Aceitar tanto vetores literais quanto símbolos (variáveis)
				if dataNode.Type == reader.NodeVector || dataNode.Type == reader.NodeSymbol {
					message = "Using 'apply hash-map' on a vector is unnecessary. Use a map literal or proper map construction instead."
					report = true
				}
			}
		}
		// Detectar apply concat (use mapcat)
		if len(node.Children) >= 3 {
			funcAppliedNode := node.Children[1]
			if funcAppliedNode.Type == reader.NodeSymbol && funcAppliedNode.Value == "concat" {
				message = "Using 'apply concat' is inefficient. Consider using 'mapcat' or 'reduce into' for better performance."
				report = true
			}
		}
	case "concat":
		// Detectar concat excessivo
		if r.hasNestedConcat(node) {
			message = "Nested 'concat' operations can be inefficient and create deep call stacks. Consider using 'into' with multiple collections or 'transduce' with 'cat'."
			report = true
		}
	case "reverse":
		// Detectar reverse em lazy sequences
		if len(node.Children) >= 2 {
			collectionArg := node.Children[1]
			if collectionArg.Type == reader.NodeList && len(collectionArg.Children) >= 2 {
				innerFunc := collectionArg.Children[0]
				if innerFunc.Type == reader.NodeSymbol {
					// Detectar reverse em map/filter/etc (força realização)
					if innerFunc.Value == "map" || innerFunc.Value == "filter" || innerFunc.Value == "remove" || innerFunc.Value == "take" || innerFunc.Value == "drop" {
						message = "Using 'reverse' on a lazy sequence forces full realization. Consider alternative approaches or 'into' with reversed accumulator."
						report = true
					}
				}
			}
		}
	case "merge":
		// Detectar merge com muitos argumentos pequenos
		if len(node.Children) > 4 {
			message = "Using 'merge' with many small maps can be inefficient. Consider using 'reduce-kv' or 'into' for better performance."
			report = true
		}
	case "assoc-in":
		// Detectar assoc-in desnecessário para um nível
		if len(node.Children) >= 4 {
			keyPathNode := node.Children[2]
			if keyPathNode.Type == reader.NodeVector && len(keyPathNode.Children) == 1 {
				message = "Using 'assoc-in' with a single key is unnecessary overhead. Use 'assoc' instead."
				report = true
			}
		}
	case "get-in":
		// Detectar get-in desnecessário para um nível
		if len(node.Children) >= 3 {
			keyPathNode := node.Children[2]
			if keyPathNode.Type == reader.NodeVector && len(keyPathNode.Children) == 1 {
				message = "Using 'get-in' with a single key is unnecessary overhead. Use 'get' instead."
				report = true
			}
		}
	case "zipmap":
		// Detectar zipmap com range (use map-indexed)
		if len(node.Children) >= 3 {
			keysArg := node.Children[1]
			if keysArg.Type == reader.NodeList && len(keysArg.Children) >= 1 {
				if keysArg.Children[0].Type == reader.NodeSymbol && keysArg.Children[0].Value == "range" {
					message = "Using 'zipmap' with 'range' is inefficient. Consider using 'map-indexed' with 'vector' or other alternatives."
					report = true
				}
			}
		}
	case "repeatedly":
		// Detectar repeatedly + take quando range seria melhor
		if len(node.Children) >= 2 {
			// Procurar por take aplicado ao repeatedly
			funcArg := node.Children[1]
			if funcArg.Type == reader.NodeFnLiteral {
				// Isso pode ser parte de um pattern (take n (repeatedly f))
				// Vamos detectar isso no contexto pai
			}
		}
	case "take":
		// Detectar take + repeatedly pattern
		if len(node.Children) >= 3 {
			collectionArg := node.Children[2]
			if collectionArg.Type == reader.NodeList && len(collectionArg.Children) >= 2 {
				if collectionArg.Children[0].Type == reader.NodeSymbol && collectionArg.Children[0].Value == "repeatedly" {
					message = "Using '(take n (repeatedly f))' may be less clear than using 'map' with 'range' for finite sequences."
					report = true
				}
			}
		}
	case "for":
		// Detectar for trivial quando map seria melhor
		if len(node.Children) >= 3 {
			bindingNode := node.Children[1]
			if bindingNode.Type == reader.NodeVector && len(bindingNode.Children) == 2 {
				// Pattern: (for [x xs] (f x))
				bodyNode := node.Children[2]
				if bodyNode.Type == reader.NodeList && len(bodyNode.Children) >= 2 {
					// Verificar se o body é uma simples aplicação de função
					funcInBody := bodyNode.Children[0]
					if funcInBody.Type == reader.NodeSymbol {
						// Isso pode ser um for trivial
						message = "Simple 'for' comprehension can be replaced with 'map' for better clarity and performance."
						report = true
					}
				}
			}
		}
	case "filter":
		// Detectar filter com not quando remove seria melhor
		if len(node.Children) >= 2 {
			predNode := node.Children[1]
			if predNode.Type == reader.NodeList && len(predNode.Children) >= 2 {
				if predNode.Children[0].Type == reader.NodeSymbol && predNode.Children[0].Value == "not" {
					message = "Using 'filter' with 'not' is less clear than using 'remove' directly."
					report = true
				}
			}
			// Detectar filter com (comp not predicate) - novo padrão
			if predNode.Type == reader.NodeList && len(predNode.Children) >= 2 {
				if predNode.Children[0].Type == reader.NodeSymbol && predNode.Children[0].Value == "comp" {
					if len(predNode.Children) >= 2 && predNode.Children[1].Type == reader.NodeSymbol && predNode.Children[1].Value == "not" {
						message = "Using 'filter' with '(comp not predicate)' is less clear than using 'remove' directly."
						report = true
					}
				}
			}
		}

	case "remove":
		// Detectar remove com not (double negation)
		if len(node.Children) >= 2 {
			predNode := node.Children[1]
			if predNode.Type == reader.NodeList && len(predNode.Children) >= 2 {
				if predNode.Children[0].Type == reader.NodeSymbol && predNode.Children[0].Value == "not" {
					message = "Using 'remove' with 'not' creates double negation. Use 'filter' instead."
					report = true
				}
			}
			// Detectar remove com (comp not predicate) - novo padrão
			if predNode.Type == reader.NodeList && len(predNode.Children) >= 2 {
				if predNode.Children[0].Type == reader.NodeSymbol && predNode.Children[0].Value == "comp" {
					if len(predNode.Children) >= 2 && predNode.Children[1].Type == reader.NodeSymbol && predNode.Children[1].Value == "not" {
						message = "Using 'remove' with '(comp not predicate)' creates double negation. Use 'filter' instead."
						report = true
					}
				}
			}
		}
	case "into":
		// Detectar into [] xs quando vec seria melhor
		if len(node.Children) >= 3 {
			targetNode := node.Children[1]
			if targetNode.Type == reader.NodeVector && len(targetNode.Children) == 0 {
				// Verificar se não é um contexto de transdução (into [] (comp ...) ...)
				isTransduction := false
				if len(node.Children) >= 3 {
					secondArg := node.Children[2]
					if secondArg.Type == reader.NodeList && len(secondArg.Children) >= 2 {
						if secondArg.Children[0].Type == reader.NodeSymbol && secondArg.Children[0].Value == "comp" {
							isTransduction = true
						}
					}
				}

				if !isTransduction {
					message = "Using 'into []' is less clear than using 'vec' for converting to vector."
					report = true
				}
			}
		}
		// Detectar into #{} xs quando set seria melhor
		if len(node.Children) >= 3 {
			targetNode := node.Children[1]
			if targetNode.Type == reader.NodeSet && len(targetNode.Children) == 0 {
				message = "Using 'into #{}' is less clear than using 'set' for converting to set."
				report = true
			}
		}
	case "doall":
		// Detectar doall com map (perigoso em produção)
		if len(node.Children) >= 2 {
			collectionArg := node.Children[1]
			if collectionArg.Type == reader.NodeList && len(collectionArg.Children) >= 2 {
				if collectionArg.Children[0].Type == reader.NodeSymbol && collectionArg.Children[0].Value == "map" {
					message = "Using 'doall' with 'map' forces full realization and should be avoided in production. Consider using 'mapv' or 'transduce'."
					report = true
				}
			}
		}
	case "=":
		// Detectar (= 0 (count coll)) quando empty? seria melhor
		if len(node.Children) >= 3 {
			firstArg := node.Children[1]
			secondArg := node.Children[2]

			if firstArg.Type == reader.NodeNumber && firstArg.Value == "0" {
				if secondArg.Type == reader.NodeList && len(secondArg.Children) >= 2 {
					if secondArg.Children[0].Type == reader.NodeSymbol && secondArg.Children[0].Value == "count" {
						message = "Using '(= 0 (count coll))' is less idiomatic than using 'empty?' and may be less efficient."
						report = true
					}
				}
			}
		}
	case ">":
		// Detectar (> (count coll) 0) quando seq seria melhor
		if len(node.Children) >= 3 {
			firstArg := node.Children[1]
			secondArg := node.Children[2]

			if firstArg.Type == reader.NodeList && len(firstArg.Children) >= 2 {
				if firstArg.Children[0].Type == reader.NodeSymbol && firstArg.Children[0].Value == "count" {
					if secondArg.Type == reader.NodeNumber && secondArg.Value == "0" {
						message = "Using '(> (count coll) 0)' is less idiomatic than using 'seq' and may be less efficient."
						report = true
					}
				}
			}
		}
	case "not":
		// Detectar (not (empty? coll)) quando seq seria melhor
		if len(node.Children) >= 2 {
			innerNode := node.Children[1]
			if innerNode.Type == reader.NodeList && len(innerNode.Children) >= 2 {
				if innerNode.Children[0].Type == reader.NodeSymbol && innerNode.Children[0].Value == "empty?" {
					message = "Using '(not (empty? coll))' is less idiomatic than using 'seq'. The docstring of 'empty?' suggests using 'seq' instead."
					report = true
				}
				// Detectar (not (zero? (count coll))) quando seq seria melhor
				if innerNode.Children[0].Type == reader.NodeSymbol && innerNode.Children[0].Value == "zero?" {
					if len(innerNode.Children) >= 2 {
						zeroArg := innerNode.Children[1]
						if zeroArg.Type == reader.NodeList && len(zeroArg.Children) >= 2 {
							if zeroArg.Children[0].Type == reader.NodeSymbol && zeroArg.Children[0].Value == "count" {
								message = "Using '(not (zero? (count coll)))' is less idiomatic than using 'seq' and may be less efficient."
								report = true
							}
						}
					}
				}
			}
		}
	case "keys":
		// Detectar keys + group-by quando distinct seria melhor
		if len(node.Children) >= 2 {
			collectionArg := node.Children[1]
			if collectionArg.Type == reader.NodeList && len(collectionArg.Children) >= 2 {
				if collectionArg.Children[0].Type == reader.NodeSymbol && collectionArg.Children[0].Value == "group-by" {
					message = "Using 'keys' on 'group-by' result just to get distinct values is inefficient. Use 'distinct' on the original collection instead."
					report = true
				}
			}
		}
	case "seq":
		// Detectar seq + count quando empty? seria melhor
		if len(node.Children) >= 2 {
			collectionArg := node.Children[1]
			if collectionArg.Type == reader.NodeList && len(collectionArg.Children) >= 2 {
				if collectionArg.Children[0].Type == reader.NodeSymbol && collectionArg.Children[0].Value == "count" {
					message = "Using 'seq' with 'count' is redundant. Use 'seq' directly on the collection."
					report = true
				}
			}
		}
	}

	if report {
		meta := r.Meta()
		return &Finding{
			RuleID:   meta.ID,
			Message:  message,
			Filepath: filepath,
			Location: node.Location,
			Severity: meta.Severity,
		}
	}

	return nil
}

// hasNestedConcat detecta concat aninhados
func (r *InappropriateCollectionRule) hasNestedConcat(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}

	if funcNode := node.Children[0]; funcNode.Type == reader.NodeSymbol && funcNode.Value == "concat" {
		// Verificar se algum argumento é outro concat
		for i := 1; i < len(node.Children); i++ {
			arg := node.Children[i]
			if arg.Type == reader.NodeList && len(arg.Children) > 0 {
				if innerFunc := arg.Children[0]; innerFunc.Type == reader.NodeSymbol && innerFunc.Value == "concat" {
					return true
				}
			}
		}
	}
	return false
}

// checkForApplyHashMap verifica recursivamente se há apply hash-map na expressão
func (r *InappropriateCollectionRule) checkForApplyHashMap(node *reader.RichNode, filepath string) *Finding {
	if node.Type == reader.NodeList && len(node.Children) >= 3 {
		if funcNode := node.Children[0]; funcNode.Type == reader.NodeSymbol && funcNode.Value == "apply" {
			if len(node.Children) >= 3 {
				funcAppliedNode := node.Children[1]
				dataNode := node.Children[2]
				if funcAppliedNode.Type == reader.NodeSymbol && funcAppliedNode.Value == "hash-map" {
					// Aceitar tanto vetores literais quanto símbolos (variáveis)
					if dataNode.Type == reader.NodeVector || dataNode.Type == reader.NodeSymbol {
						meta := r.Meta()
						return &Finding{
							RuleID:   meta.ID,
							Message:  "Using 'apply hash-map' on a vector is unnecessary. Use a map literal or proper map construction instead.",
							Filepath: filepath,
							Location: node.Location,
							Severity: meta.Severity,
						}
					}
				}
			}
		}
	}

	// Verificar recursivamente nos filhos
	for _, child := range node.Children {
		if finding := r.checkForApplyHashMap(child, filepath); finding != nil {
			return finding
		}
	}

	return nil
}

func init() {
	RegisterRule(&InappropriateCollectionRule{})
}
