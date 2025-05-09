package rules

import (
	"fmt"
	"os"

	"github.com/thlaurentino/arit/internal/reader"
)

type InappropriateCollectionRule struct{}

func (r *InappropriateCollectionRule) Meta() Rule {
	return Rule{
		ID:          "inappropriate-collection",
		Name:        "Inappropriate Collection Usage",
		Description: "Detects inefficient or non-idiomatic usage of collection functions. This includes using 'last' on vectors (consider 'peek'), 'nth' on lists for random access, or 'contains?' on lists/vectors for key lookups instead of sets/maps.",
		Severity:    SeverityHint,
	}
}

func getCollectionNodeType(node *reader.RichNode) reader.NodeType {
	fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType ENTER] Node Type: %s, Node Value: %s, Node Location: %v\n", node.Type, node.Value, node.Location)
	if node == nil {
		fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Node is nil, returning Unknown\n")
		return reader.NodeUnknown
	}

	if node.Type == reader.NodeVector || node.Type == reader.NodeList || node.Type == reader.NodeMap || node.Type == reader.NodeSet {
		fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Node is a literal collection %s\n", node.Type)
		return node.Type
	}

	if node.Type == reader.NodeSymbol {
		fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] Node is Symbol. Checking ResolvedDefinition.\n")
		if node.ResolvedDefinition != nil {
			defNode := node.ResolvedDefinition
			fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] ResolvedDefinition exists. defNode Type: %s, defNode Value: %s, defNode Children: %d, defNode Location: %v\n", defNode.Type, defNode.Value, len(defNode.Children), defNode.Location)

			if defNode.Type == reader.NodeVector || defNode.Type == reader.NodeMap || defNode.Type == reader.NodeSet {
				fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] ResolvedDefinition is a direct literal Vector/Map/Set: %s\n", defNode.Type)
				return defNode.Type
			}

			if defNode.Type == reader.NodeList && len(defNode.Children) > 0 {
				defTypeSymbolNode := defNode.Children[0]
				fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] ResolvedDefinition is a List. First child (def type symbol) Type: %s, Value: %s\n", defTypeSymbolNode.Type, defTypeSymbolNode.Value)

				if defTypeSymbolNode.Type == reader.NodeSymbol {
					defFormName := defTypeSymbolNode.Value
					if (defFormName == "def" || defFormName == "defonce") && len(defNode.Children) >= 3 {
						valueNode := defNode.Children[len(defNode.Children)-1]
						fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] Found '%s'. valueNode Type: %s, valueNode Value: %s, valueNode Location: %v\n", defFormName, valueNode.Type, valueNode.Value, valueNode.Location)

						if valueNode.Type == reader.NodeList && len(valueNode.Children) > 0 && valueNode.Children[0].Type == reader.NodeSymbol {
							constructorFuncNode := valueNode.Children[0]
							constructorFuncName := constructorFuncNode.Value
							fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] ValueNode is a function call: %s\n", constructorFuncName)

							switch constructorFuncName {
							case "vector", "vec":
								fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from constructor '%s' to Vector\n", constructorFuncName)
								return reader.NodeVector
							case "list", "list*":
								fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from constructor '%s' to List\n", constructorFuncName)
								return reader.NodeList
							case "hash-map", "array-map", "sorted-map", "sorted-map-by":
								fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from constructor '%s' to Map\n", constructorFuncName)
								return reader.NodeMap
							case "map":
								fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from constructor '%s' (map f coll) to List/Sequence\n", constructorFuncName)
								return reader.NodeList
							case "hash-set", "set", "sorted-set", "sorted-set-by":
								fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from constructor '%s' to Set\n", constructorFuncName)
								return reader.NodeSet
							case "into":
								if len(valueNode.Children) > 1 {
									targetCollectionNode := valueNode.Children[1]
									fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType into] Target Collection Node for 'into': Type: %s, Value: %s, Location: %v\n", targetCollectionNode.Type, targetCollectionNode.Value, targetCollectionNode.Location)

									if targetCollectionNode.Type == reader.NodeMap {
										fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from 'into {}' to Map\n")
										return reader.NodeMap
									}
									if targetCollectionNode.Type == reader.NodeSet {
										fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from 'into #{...}' to Set\n")
										return reader.NodeSet
									}
									if targetCollectionNode.Type == reader.NodeVector {
										fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from 'into []' to Vector\n")
										return reader.NodeVector
									}
									if targetCollectionNode.Type == reader.NodeList {
										fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from 'into () or (list)..' to List\n")
										return reader.NodeList
									}

									if targetCollectionNode.Type == reader.NodeSymbol {
										fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType into] Target for 'into' is a symbol. Attempting recursive call to getCollectionNodeType for target: %s\n", targetCollectionNode.Value)
										resolvedTargetType := getCollectionNodeType(targetCollectionNode)
										if resolvedTargetType != reader.NodeUnknown {
											fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from 'into [resolved_symbol]' to %s\n", resolvedTargetType)
											return resolvedTargetType
										}
									}
									fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType into] Target for 'into' (%s) is not a resolvable literal or symbol. Defaulting to List for 'into' result.\n", targetCollectionNode.Type)
									return reader.NodeList
								} else {
									fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType into] 'into' call has insufficient arguments. Defaulting to Unknown.\n")
									return reader.NodeUnknown
								}
							default:

								fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] Constructor function '%s' is a NodeList but not in switch. Treating as generic List/Sequence.\n", constructorFuncName)
								return reader.NodeList
							}
						}

						if valueNode.Type == reader.NodeVector || valueNode.Type == reader.NodeMap || valueNode.Type == reader.NodeSet {
							fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from '%s' to literal collection %s (Vector, Map, or Set)\n", defFormName, valueNode.Type)
							return valueNode.Type
						}

						if valueNode.Type == reader.NodeList {
							fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Resolved from '%s' to literal List or unhandled fn call: %s\n", defFormName, valueNode.Type)
							return reader.NodeList
						}

						fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] ValueNode in '%s' is not a literal or recognized function call producing a collection. Type: %s. Defaulting to Unknown.\n", defFormName, valueNode.Type)

					} else {
						fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] ResolvedDefinition is not a List or has no children. Type: %s\n", defNode.Type)
					}
				} else {
					fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] First child of ResolvedDefinition List is not a Symbol. Type: %s\n", defTypeSymbolNode.Type)
				}
			} else {
				fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] ResolvedDefinition is not a List or has no children. Type: %s\n", defNode.Type)
			}
		} else {
			fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType] Node is Symbol but ResolvedDefinition is nil.\n")
		}
	}
	fmt.Fprintf(os.Stderr, "[DEBUG getCollectionNodeType EXIT] Defaulting to Unknown for Node Type %s\n", node.Type)
	return reader.NodeUnknown
}

func (r *InappropriateCollectionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	fmt.Fprintf(os.Stderr, "[DEBUG InappropriateCollectionRule ENTER] NodeType: %s, Location: %v\n", node.Type, node.Location)

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

	fmt.Fprintf(os.Stderr, "[DEBUG InappropriateCollectionRule INFO] FuncName: %s, CollectionNodeType: %s, CollectionTypeFromFunc: %s, Location: %v\n", funcName, collectionNode.Type, collectionType, collectionNode.Location)

	var message string
	report := false

	switch funcName {
	case "last":
		fmt.Fprintf(os.Stderr, "[DEBUG InappropriateCollectionRule CASE LAST] CollectionType: %s for (last %s), CollectionLocation: %v\n", collectionType, collectionNode.Type, collectionNode.Location)
		if collectionType == reader.NodeVector {
			message = "Using 'last' on a vector can be inefficient for large vectors. Consider 'peek' or rethink data structure if last-element access is frequent."
			report = true
		}
	case "nth":
		fmt.Fprintf(os.Stderr, "[DEBUG InappropriateCollectionRule CASE NTH] CollectionType: %s for (nth %s ...), CollectionLocation: %v\n", collectionType, collectionNode.Type, collectionNode.Location)
		if collectionType == reader.NodeList {
			message = "Using 'nth' for random access on a list is inefficient (linear time). Use a vector if random access is needed."
			report = true
		}
	case "contains?":
		fmt.Fprintf(os.Stderr, "[DEBUG InappropriateCollectionRule CASE CONTAINS?] CollectionType: %s for (contains? %s ...), CollectionLocation: %v\n", collectionType, collectionNode.Type, collectionNode.Location)
		if collectionType == reader.NodeVector || collectionType == reader.NodeList {
			if len(node.Children) > 2 {
				keyNode := node.Children[2]
				fmt.Fprintf(os.Stderr, "[DEBUG InappropriateCollectionRule CONTAINS? KEY_NODE] KeyNodeType: %s, KeyNodeValue: %s, KeyLocation: %v\n", keyNode.Type, keyNode.Value, keyNode.Location)
				if keyNode.Type == reader.NodeKeyword || keyNode.Type == reader.NodeString || keyNode.Type == reader.NodeSymbol {
					message = fmt.Sprintf("Using 'contains?' with a non-numeric key on a %s is inefficient. Use a set for presence checks or a map for key lookups.", collectionType)
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

func init() {
	RegisterRule(&InappropriateCollectionRule{})
}
