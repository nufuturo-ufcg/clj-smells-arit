package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type OverengineeringCoreAsyncRule struct{ rules.Rule }

func (r *OverengineeringCoreAsyncRule) Meta() rules.Rule { return r.Rule }

func coreAsyncOperation(node *reader.RichNode) string {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 ||
		node.Children[0].Type != reader.NodeSymbol {
		return ""
	}
	if node.Children[0].Resolution != nil && node.Children[0].Resolution.CanonicalName != "" {
		return node.Children[0].Resolution.CanonicalName
	}
	return node.Children[0].Value
}

func coreAsyncHasSuffix(symbol string, names ...string) bool {
	for _, name := range names {
		if symbol == name || strings.HasSuffix(symbol, "/"+name) {
			return true
		}
	}
	return false
}

func coreAsyncIsOperation(node *reader.RichNode, canonicalNames ...string) bool {
	resolved := rules.ResolvedCall(node)
	if resolved == nil || resolved.Kind == reader.ResolutionLocal {
		return false
	}
	for _, canonical := range canonicalNames {
		if resolved.Kind != reader.ResolutionUnresolved && resolved.CanonicalName == canonical {
			return true
		}
		if resolved.Kind == reader.ResolutionUnresolved && strings.HasPrefix(canonical, "clojure.core/") &&
			len(node.Children) > 0 && node.Children[0].Value == strings.TrimPrefix(canonical, "clojure.core/") {
			return true
		}
	}
	return false
}

func isCallbackFunction(node *reader.RichNode, parent *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if coreAsyncIsOperation(node, "clojure.core/fn", "clojure.core/fn*") {
		if parent != nil {
			parentOp := coreAsyncOperation(parent)
			if !coreAsyncHasSuffix(parentOp, "go", "thread", "future") {
				return true
			}
		}
	}
	return false
}

func coreAsyncCountSingleValuePuts(node *reader.RichNode, parent *reader.RichNode, channel string) (int, bool) {
	if node == nil {
		return 0, false
	}
	if isCallbackFunction(node, parent) {
		return 0, true
	}
	if coreAsyncIsOperation(node,
		"clojure.core/loop", "clojure.core.async/go-loop", "clojure.core/doseq", "clojure.core.async/pipeline",
		"clojure.core.async/pipeline-blocking", "clojure.core.async/pipeline-async", "clojure.core.async/mult",
		"clojure.core.async/pub", "clojure.core/while", "clojure.core/proxy", "clojure.core/reify") {
		return 0, true
	}
	count := 0
	if coreAsyncIsOperation(node, "clojure.core.async/>!", "clojure.core.async/>!!", "clojure.core.async/put!") && len(node.Children) >= 3 &&
		node.Children[1].Type == reader.NodeSymbol && node.Children[1].Value == channel {
		count++
	}
	for _, child := range node.Children {
		childCount, complex := coreAsyncCountSingleValuePuts(child, node, channel)
		count += childCount
		if complex {
			return count, true
		}
	}
	return count, false
}

func coreAsyncSingleValueChannel(node *reader.RichNode) (*reader.RichNode, string) {
	if !coreAsyncIsOperation(node, "clojure.core/let") || len(node.Children) < 4 || node.Children[1].Type != reader.NodeVector {
		return nil, ""
	}
	bindings := node.Children[1]
	for index := 0; index+1 < len(bindings.Children); index += 2 {
		name, value := bindings.Children[index], bindings.Children[index+1]
		if name.Type != reader.NodeSymbol || !coreAsyncIsOperation(value, "clojure.core.async/chan") {
			continue
		}
		last := node.Children[len(node.Children)-1]
		if last.Type != reader.NodeSymbol || last.Value != name.Value {
			continue
		}
		puts, complex := coreAsyncCountSingleValuePuts(node, nil, name.Value)
		if puts != 1 || complex {
			continue
		}
		return value, name.Value
	}
	return nil, ""
}

func (r *OverengineeringCoreAsyncRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if strings.HasSuffix(filepath, ".cljs") || strings.HasSuffix(filepath, "user.clj") {
		return nil
	}
	value, channel := coreAsyncSingleValueChannel(node)
	if value != nil {
		return rules.SetContextualFindingWithEvidence(&rules.Finding{
			RuleID: r.ID, Filepath: filepath, Location: value.Location,
			Severity: rules.ContextualSeverity(context, r.Severity), Tags: rules.ContextualTags(context),
			Message: fmt.Sprintf("Channel %q is used only to return one value; prefer a direct value, future, or promise.", channel),
		}, "A single-value channel may be a deliberate part of the API's asynchronous contract.", "async-api-contract", "lifecycle-contract")
	}
	return nil
}

func init() {
	rules.RegisterRule(&OverengineeringCoreAsyncRule{Rule: rules.Rule{
		ID: "overengineering-with-core-async", Name: "Overengineering with core.async",
		Description: "Detects a channel allocated, written once, and directly returned as a single-value result.",
		Severity:    rules.SeverityHint,
	}})
}
