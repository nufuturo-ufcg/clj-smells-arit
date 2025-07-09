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

	if node.Type == reader.NodeVector || node.Type == reader.NodeList || node.Type == reader.NodeMap || node.Type == reader.NodeSet {
		return node.Type
	}

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

func detectQuotedList(node *reader.RichNode) bool {

	if node.Type == reader.NodeList && len(node.Children) == 2 {
		if firstChild := node.Children[0]; firstChild.Type == reader.NodeSymbol && firstChild.Value == "quote" {
			if secondChild := node.Children[1]; secondChild.Type == reader.NodeList {
				return true
			}
		}
	}

	if node.Type == reader.NodeQuote {
		if len(node.Children) > 0 && node.Children[0].Type == reader.NodeList {
			return true
		}
	}

	return false
}

func (r *InappropriateCollectionRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *Finding {
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

		if detectQuotedList(collectionNode) {
			message = "Using 'nth' for random access on a list is inefficient (linear time). Use a vector if random access is needed."
			report = true
		} else if collectionType == reader.NodeList {
			message = "Using 'nth' for random access on a list is inefficient (linear time). Use a vector if random access is needed."
			report = true
		} else if collectionNode.Type == reader.NodeQuote {

			message = "Using 'nth' for random access on a list is inefficient (linear time). Use a vector if random access is needed."
			report = true
		}
	case "some":

		if len(node.Children) >= 3 {
			predNode := node.Children[1]
			targetCollectionNode := node.Children[2]

			if predNode.Type == reader.NodeFnLiteral && targetCollectionNode.Type == reader.NodeVector {
				message = "Using 'some' for membership check on a vector is inefficient. Use a set with 'contains?' for O(1) membership testing."
				report = true
			}
		}
	case "first":

		if len(node.Children) >= 2 && collectionNode.Type == reader.NodeList && len(collectionNode.Children) >= 2 {
			if collectionNode.Children[0].Type == reader.NodeSymbol && collectionNode.Children[0].Value == "filter" {
				message = "Using '(first (filter ...))' is inefficient. Consider using 'some' which stops after the first match."
				report = true
			}
		}
	case "empty?":

		if len(node.Children) >= 2 && collectionNode.Type == reader.NodeList && len(collectionNode.Children) >= 2 {
			if collectionNode.Children[0].Type == reader.NodeSymbol && collectionNode.Children[0].Value == "filter" {
				message = "Using '(empty? (filter ...))' is inefficient. Consider using 'not-any?' for early termination."
				report = true
			}
		}
	case "count":

		if len(node.Children) >= 2 && collectionNode.Type == reader.NodeList && len(collectionNode.Children) >= 2 {
			if collectionNode.Children[0].Type == reader.NodeSymbol && collectionNode.Children[0].Value == "filter" {
				message = "Using '(count (filter ...))' processes entire collection. Consider 'transduce' with counting reducer for potentially better performance."
				report = true
			}
		}
	case "sequence":

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

		if len(node.Children) >= 3 {
			accumInitNode := node.Children[2]
			if detectQuotedList(accumInitNode) || accumInitNode.Type == reader.NodeQuote {
				message = "Using 'reduce' with a list as accumulator can be inefficient. Consider using a vector for better performance."
				report = true
			}
		}
	case "map":

		if len(node.Children) >= 3 {
			targetCollectionNode := node.Children[len(node.Children)-1]
			if targetCollectionNode.Type == reader.NodeMap {
				message = "Using 'map' on a hash-map doesn't preserve order. Consider using a vector of pairs or sorted-map if order matters."
				report = true
			}
		}

		if len(node.Children) >= 2 {
			funcArg := node.Children[1]
			if funcArg.Type == reader.NodeSymbol && funcArg.Value == "identity" {
				message = "Using 'map' with 'identity' is unnecessary. Consider using 'seq' or removing the transformation entirely."
				report = true
			}
		}
	case "apply":
		if len(node.Children) >= 3 {
			funcAppliedNode := node.Children[1]
			if funcAppliedNode.Type == reader.NodeSymbol {
				switch funcAppliedNode.Value {
				case "hash-map":
					dataNode := node.Children[2]
					if dataNode.Type == reader.NodeVector || dataNode.Type == reader.NodeSymbol {
						message = "Using 'apply hash-map' on a vector is unnecessary. Use a map literal or proper map construction instead."
						report = true
					}
				case "concat":
					message = "Using 'apply concat' is inefficient. Consider using 'mapcat' or 'reduce into' for better performance."
					report = true
				}
			}
		}
	case "concat":

		if r.hasNestedConcat(node) {
			message = "Nested 'concat' operations can be inefficient and create deep call stacks. Consider using 'into' with multiple collections or 'transduce' with 'cat'."
			report = true
		}
	case "reverse":

		if len(node.Children) >= 2 {
			collectionArg := node.Children[1]
			if collectionArg.Type == reader.NodeList && len(collectionArg.Children) >= 2 {
				innerFunc := collectionArg.Children[0]
				if innerFunc.Type == reader.NodeSymbol {

					if innerFunc.Value == "map" || innerFunc.Value == "filter" || innerFunc.Value == "remove" || innerFunc.Value == "take" || innerFunc.Value == "drop" {
						message = "Using 'reverse' on a lazy sequence forces full realization. Consider alternative approaches or 'into' with reversed accumulator."
						report = true
					}
				}
			}
		}
	case "merge":

		if len(node.Children) > 4 {
			message = "Using 'merge' with many small maps can be inefficient. Consider using 'reduce-kv' or 'into' for better performance."
			report = true
		}
	case "assoc-in":

		if len(node.Children) >= 4 {
			keyPathNode := node.Children[2]
			if keyPathNode.Type == reader.NodeVector && len(keyPathNode.Children) == 1 {
				message = "Using 'assoc-in' with a single key is unnecessary overhead. Use 'assoc' instead."
				report = true
			}
		}
	case "get-in":

		if len(node.Children) >= 3 {
			keyPathNode := node.Children[2]
			if keyPathNode.Type == reader.NodeVector && len(keyPathNode.Children) == 1 {
				message = "Using 'get-in' with a single key is unnecessary overhead. Use 'get' instead."
				report = true
			}
		}
	case "zipmap":

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
		// Note: This case is intentionally minimal as most repeatedly usage patterns are acceptable
		// Could be extended in the future to detect specific anti-patterns
		if len(node.Children) >= 2 {
			funcArg := node.Children[1]
			if funcArg.Type == reader.NodeFnLiteral {
				// Check for simple cases that could be replaced with other functions
				// This is a placeholder for future enhancements
			}
		}
	case "take":

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

		if len(node.Children) >= 3 {
			bindingNode := node.Children[1]
			if bindingNode.Type == reader.NodeVector && len(bindingNode.Children) == 2 {

				bodyNode := node.Children[2]
				if bodyNode.Type == reader.NodeList && len(bodyNode.Children) >= 2 {

					funcInBody := bodyNode.Children[0]
					if funcInBody.Type == reader.NodeSymbol {

						message = "Simple 'for' comprehension can be replaced with 'map' for better clarity and performance."
						report = true
					}
				}
			}
		}
	case "filter":

		if len(node.Children) >= 2 {
			predNode := node.Children[1]
			if predNode.Type == reader.NodeList && len(predNode.Children) >= 2 {
				if predNode.Children[0].Type == reader.NodeSymbol && predNode.Children[0].Value == "not" {
					message = "Using 'filter' with 'not' is less clear than using 'remove' directly."
					report = true
				}
			}

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

		if len(node.Children) >= 2 {
			predNode := node.Children[1]
			if predNode.Type == reader.NodeList && len(predNode.Children) >= 2 {
				if predNode.Children[0].Type == reader.NodeSymbol && predNode.Children[0].Value == "not" {
					message = "Using 'remove' with 'not' creates double negation. Use 'filter' instead."
					report = true
				}
			}

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

		if len(node.Children) >= 3 {
			targetNode := node.Children[1]
			if targetNode.Type == reader.NodeVector && len(targetNode.Children) == 0 {

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

		if len(node.Children) >= 3 {
			targetNode := node.Children[1]
			if targetNode.Type == reader.NodeSet && len(targetNode.Children) == 0 {
				message = "Using 'into #{}' is less clear than using 'set' for converting to set."
				report = true
			}
		}
	case "doall":

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

		if len(node.Children) >= 2 {
			innerNode := node.Children[1]
			if innerNode.Type == reader.NodeList && len(innerNode.Children) >= 2 {
				if innerNode.Children[0].Type == reader.NodeSymbol && innerNode.Children[0].Value == "empty?" {
					message = "Using '(not (empty? coll))' is less idiomatic than using 'seq'. The docstring of 'empty?' suggests using 'seq' instead."
					report = true
				}

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

func (r *InappropriateCollectionRule) hasNestedConcat(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}

	if funcNode := node.Children[0]; funcNode.Type == reader.NodeSymbol && funcNode.Value == "concat" {
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

func init() {
	RegisterRule(&InappropriateCollectionRule{})
}
