package clojurespecific

import (
	"fmt"
	"github.com/thlaurentino/arit/internal/rules"

	"github.com/thlaurentino/arit/internal/reader"
)

type NonIdiomaticRecordConstructionRule struct {
	rules.Rule
}

func (r *NonIdiomaticRecordConstructionRule) Meta() rules.Rule {
	return r.Rule
}

func (r *NonIdiomaticRecordConstructionRule) isRecordSymbol(name string, recordFuncs []string) (string, bool) {
	for _, function := range recordFuncs {
		if name == function {
			return function, true
		}
	}
	return "", false
}

func (r *NonIdiomaticRecordConstructionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if r.IsInside(context, "__non-evaluated__", "comment") ||
		nonIdiomaticRecordIsInsideMacroDefinition(context) {
		return nil
	}
	var recordFuncs []string
	if rf, ok := context["recordFunctions"].([]string); ok {
		recordFuncs = rf
	}
	recordFieldCounts, _ := context["recordFieldCounts"].(map[string]int)
	if recordFieldCounts == nil {
		recordFieldCounts = map[string]int{}
		context["recordFieldCounts"] = recordFieldCounts
	}

	if node.Type != reader.NodeList || len(node.Children) <= 0 || node.Children[0].Type != reader.NodeSymbol {
		return nil
	}

	firstChild := node.Children[0].Value
	if firstChild == "ns" {
		context["recordFunctions"] = []string{}
		context["recordFieldCounts"] = map[string]int{}
		return nil
	}

	if firstChild == "defrecord" && len(node.Children) > 2 && node.Children[1].Type == reader.NodeSymbol && node.Children[2].Type == reader.NodeVector {
		recordFuncs = append(recordFuncs, node.Children[1].Value)
		context["recordFunctions"] = recordFuncs
		recordFieldCounts[node.Children[1].Value] = len(node.Children[2].Children)
		return nil
	}

	var targetRecord string
	if len(firstChild) > 1 && firstChild[len(firstChild)-1] == '.' {
		candidate := firstChild[:len(firstChild)-1]
		for _, function := range recordFuncs {
			if candidate == function && len(node.Children)-1 == recordFieldCounts[function] {
				targetRecord = function
				break
			}
		}
	} else if firstChild == "new" && len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
		candidate := node.Children[1].Value
		for _, function := range recordFuncs {
			if candidate == function && len(node.Children)-2 == recordFieldCounts[function] {
				targetRecord = function
				break
			}
		}
	}

	if targetRecord != "" {
		return rules.SetContextualFindingWithEvidence(&rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Using Java interop syntax to instantiate the defrecord instead of ->%s or map->%s", targetRecord, targetRecord),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}, "The positional constructor may be deliberate for interoperability, performance, or field-order contracts.", "interop-contract", "field-order-contract")
	}

	return nil
}

func nonIdiomaticRecordIsInsideMacroDefinition(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		if ancestor.Children[0].Value == "defmacro" {
			return true
		}
	}
	return false
}

func init() {
	defaultRule := &NonIdiomaticRecordConstructionRule{
		Rule: rules.Rule{
			ID:          "non-idiomatic-record-construction",
			Name:        "Non-idiomatic Record Construction",
			Description: "Using Java's positional interpolate constructor to instantiate a defrecord causes the code to break silently if the fields are reordered.",
			Severity:    rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}
