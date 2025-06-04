package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type UnnecessaryIntoRule struct {
	Rule
	CheckTypeTransformations bool `json:"check_type_transformations" yaml:"check_type_transformations"`
	CheckMapMapping          bool `json:"check_map_mapping" yaml:"check_map_mapping"`
	CheckTransducerAPI       bool `json:"check_transducer_api" yaml:"check_transducer_api"`
}

func (r *UnnecessaryIntoRule) Meta() Rule {
	return r.Rule
}

var typeTransformationPatterns = map[string]string{
	"[]":  "vec",
	"#{}": "set",
	"()":  "seq",
}

func isEmptyCollection(node *reader.RichNode) (bool, string) {
	if node == nil {
		return false, ""
	}

	switch node.Type {
	case reader.NodeVector:
		if len(node.Children) == 0 {
			return true, "[]"
		}
	case reader.NodeSet:
		if len(node.Children) == 0 {
			return true, "#{}"
		}
	case reader.NodeList:
		if len(node.Children) == 0 {
			return true, "()"
		}
	}

	return false, ""
}

func isMapFunction(funcName string) bool {
	mapFunctions := map[string]bool{
		"map":    true,
		"mapcat": true,
		"mapv":   true,
		"pmap":   true,
		"for":    true,
	}
	return mapFunctions[funcName]
}

func isFilterFunction(funcName string) bool {
	filterFunctions := map[string]bool{
		"filter":       true,
		"remove":       true,
		"keep":         true,
		"keep-indexed": true,
		"distinct":     true,
		"take":         true,
		"drop":         true,
		"take-while":   true,
		"drop-while":   true,
	}
	return filterFunctions[funcName]
}

func (r *UnnecessaryIntoRule) checkTypeTransformation(node *reader.RichNode) *Finding {

	if len(node.Children) == 3 {
		firstArg := node.Children[1]
		secondArg := node.Children[2]

		if isEmpty, collType := isEmptyCollection(firstArg); isEmpty {
			if replacement, exists := typeTransformationPatterns[collType]; exists {
				meta := r.Meta()
				return &Finding{
					RuleID: meta.ID,
					Message: fmt.Sprintf("Unnecessary use of 'into' for type transformation. Use '(%s %s)' instead of '(into %s %s)' for better readability and performance.",
						replacement, getNodeText(secondArg), collType, getNodeText(secondArg)),
					Filepath: "",
					Location: node.Location,
					Severity: meta.Severity,
				}
			}
		}
	}

	return nil
}

func (r *UnnecessaryIntoRule) checkMapMapping(node *reader.RichNode) *Finding {

	if len(node.Children) == 3 {
		firstArg := node.Children[1]
		secondArg := node.Children[2]

		if isEmpty, collType := isEmptyCollection(firstArg); isEmpty && collType == "{}" {

			if secondArg.Type == reader.NodeList && len(secondArg.Children) > 0 {
				if funcNode := secondArg.Children[0]; funcNode.Type == reader.NodeSymbol {
					funcName := funcNode.Value
					if isMapFunction(funcName) {
						meta := r.Meta()
						return &Finding{
							RuleID:   meta.ID,
							Message:  fmt.Sprintf("Inefficient map mapping with 'into'. Consider using 'reduce-kv' for better performance when transforming map values: (reduce-kv (fn [m k v] (assoc m k (f v))) {} m)"),
							Filepath: "",
							Location: node.Location,
							Severity: meta.Severity,
						}
					}
				}
			}
		}
	}

	return nil
}

func (r *UnnecessaryIntoRule) checkTransducerAPI(node *reader.RichNode) *Finding {

	if len(node.Children) == 3 {
		firstArg := node.Children[1]
		secondArg := node.Children[2]

		if secondArg.Type == reader.NodeList && len(secondArg.Children) >= 3 {
			if funcNode := secondArg.Children[0]; funcNode.Type == reader.NodeSymbol {
				funcName := funcNode.Value
				if isMapFunction(funcName) || isFilterFunction(funcName) {
					meta := r.Meta()
					return &Finding{
						RuleID: meta.ID,
						Message: fmt.Sprintf("Consider using transducer API for better performance. Use '(into %s (%s %s) %s)' instead of '(into %s (%s %s %s))' to leverage transducers.",
							getNodeText(firstArg), funcName, getNodeText(secondArg.Children[1]), getNodeText(secondArg.Children[2]),
							getNodeText(firstArg), funcName, getNodeText(secondArg.Children[1]), getNodeText(secondArg.Children[2])),
						Filepath: "",
						Location: node.Location,
						Severity: meta.Severity,
					}
				}
			}
		}
	}

	return nil
}

func getNodeText(node *reader.RichNode) string {
	if node == nil {
		return "nil"
	}

	switch node.Type {
	case reader.NodeSymbol, reader.NodeKeyword, reader.NodeString, reader.NodeNumber:
		return node.Value
	case reader.NodeVector:
		return "[]"
	case reader.NodeSet:
		return "#{}"
	case reader.NodeMap:
		return "{}"
	case reader.NodeList:
		if len(node.Children) == 0 {
			return "()"
		}
		return "(...)"
	default:
		return "..."
	}
}

func (r *UnnecessaryIntoRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol || funcNode.Value != "into" {
		return nil
	}

	if r.CheckTypeTransformations {
		if finding := r.checkTypeTransformation(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	if r.CheckMapMapping {
		if finding := r.checkMapMapping(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	if r.CheckTransducerAPI {
		if finding := r.checkTransducerAPI(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
	}

	return nil
}

func init() {
	defaultRule := &UnnecessaryIntoRule{
		Rule: Rule{
			ID:          "unnecessary-into",
			Name:        "Unnecessary Into",
			Description: "Detects unnecessary usage of the 'into' function when more idiomatic alternatives exist. The 'into' function is useful for combining collections, but is often misused for simple type transformations like (into [] coll) instead of (vec coll), or (into #{} coll) instead of (set coll). This rule also identifies inefficient map mapping patterns and missed opportunities to use the transducer API.",
			Severity:    SeverityHint,
		},
		CheckTypeTransformations: true,
		CheckMapMapping:          true,
		CheckTransducerAPI:       true,
	}

	RegisterRule(defaultRule)
}
