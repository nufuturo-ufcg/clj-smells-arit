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
		Description: "Detects inefficient or non-idiomatic usage of collection functions. This includes using 'last' on vectors (consider 'peek'), 'nth' on lists for random access, or 'contains?' on lists/vectors for key lookups instead of sets/maps.",
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

					} else {

					}
				} else {

				}
			} else {

			}
		} else {

		}
	}

	return reader.NodeUnknown
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

		if collectionType == reader.NodeList {
			message = "Using 'nth' for random access on a list is inefficient (linear time). Use a vector if random access is needed."
			report = true
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
