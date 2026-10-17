package clojurespecific

import (
	"fmt"
	"github.com/thlaurentino/arit/internal/rules"
	"strings"

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
	if node != nil && node.Type == reader.NodeFnLiteral && len(node.Children) > 0 &&
		node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
		synthetic := &reader.RichNode{Type: reader.NodeList, Children: node.Children, Location: node.Location}
		return r.Check(synthetic, context, filepath)
	}
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
	recordPlatforms, _ := context["recordPlatforms"].(map[string]map[string]bool)
	if recordPlatforms == nil {
		recordPlatforms = map[string]map[string]bool{}
		context["recordPlatforms"] = recordPlatforms
	}
	if firstChild == "ns" {
		context["recordFunctions"] = []string{}
		context["recordFieldCounts"] = map[string]int{}
		context["recordPlatforms"] = map[string]map[string]bool{}
		return nil
	}

	if firstChild == "defrecord" && len(node.Children) > 2 && node.Children[1].Type == reader.NodeSymbol && node.Children[2].Type == reader.NodeVector {
		name := node.Children[1].Value
		recordFuncs = append(recordFuncs, name)
		context["recordFunctions"] = recordFuncs
		recordFieldCounts[name] = len(node.Children[2].Children)
		if recordPlatforms[name] == nil {
			recordPlatforms[name] = map[string]bool{}
		}
		// "" means the record exists on every platform.
		recordPlatforms[name][readerConditionalPlatform(node, context)] = true
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

	if targetRecord != "" && nonIdiomaticRecordInsideOwnDefinition(targetRecord, context) &&
		(strings.HasSuffix(filepath, ".cljs") || readerConditionalPlatform(node, context) == ":cljs") {
		// Clojure's defrecord declares ->Rec and map->Rec before the methods,
		// but ClojureScript's defines them only after the type, so a cljs
		// record's own method bodies must use the constructor.
		return nil
	}

	if targetRecord != "" {
		// In a platform branch (#?(:clj (Rec. ...))) the class only refers to
		// the defrecord when the record is defined for that platform; otherwise
		// it is another type with the same name (e.g. an imported Java class).
		if platform := readerConditionalPlatform(node, context); platform != "" {
			defined := recordPlatforms[targetRecord]
			if defined != nil && !defined[""] && !defined[platform] && !defined[":default"] {
				return nil
			}
		}
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

func nonIdiomaticRecordInsideOwnDefinition(record string, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor != nil && ancestor.Type == reader.NodeList && len(ancestor.Children) > 1 &&
			ancestor.Children[0].Type == reader.NodeSymbol && ancestor.Children[0].Value == "defrecord" &&
			ancestor.Children[1].Type == reader.NodeSymbol && ancestor.Children[1].Value == record {
			return true
		}
	}
	return false
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

// readerConditionalPlatform returns the reader-conditional feature (e.g.
// ":clj") of the innermost branch that contains node, or "" when node is not
// inside a reader conditional.
func readerConditionalPlatform(node *reader.RichNode, context map[string]interface{}) string {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for i := len(ancestors) - 1; i >= 0; i-- {
		cond := ancestors[i]
		if cond == nil || cond.Type != reader.NodeReaderCond {
			continue
		}
		branch := node
		if i+1 < len(ancestors) {
			branch = ancestors[i+1]
		}
		for j, child := range cond.Children {
			if child == branch && j > 0 && cond.Children[j-1] != nil && cond.Children[j-1].Type == reader.NodeKeyword {
				return cond.Children[j-1].Value
			}
		}
	}
	return ""
}
