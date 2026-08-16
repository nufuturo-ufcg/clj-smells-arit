package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type UnnecessaryLazinessRule struct{ rules.Rule }

func (r *UnnecessaryLazinessRule) Meta() rules.Rule { return r.Rule }

var immediatelyRealizedLazyOperations = map[string]string{
	"clojure.core/map":           "map",
	"clojure.core/filter":        "filter",
	"clojure.core/keep":          "keep",
	"clojure.core/remove":        "remove",
	"clojure.core/map-indexed":   "map-indexed",
	"clojure.core/keep-indexed":  "keep-indexed",
	"clojure.core/take":          "take",
	"clojure.core/drop":          "drop",
	"clojure.core/take-while":    "take-while",
	"clojure.core/drop-while":    "drop-while",
	"clojure.core/distinct":      "distinct",
	"clojure.core/dedupe":        "dedupe",
	"clojure.core/for":           "for",
	"clojure.core/mapcat":        "mapcat",
	"clojure.core/concat":        "concat",
	"clojure.core/flatten":       "flatten",
	"clojure.core/interleave":    "interleave",
	"clojure.core/partition":     "partition",
	"clojure.core/partition-all": "partition-all",
	"clojure.core/interpose":     "interpose",
	"clojure.core/take-nth":      "take-nth",
	"clojure.core/cycle":         "cycle",
	"clojure.core/repeat":        "repeat",
	"clojure.core/iterate":       "iterate",
}

func unnecessaryLazyChild(node *reader.RichNode) (string, bool) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) != 2 {
		return "", false
	}
	if !rules.CallResolvesTo(node, "clojure.core/vec") && !rules.CallResolvesTo(node, "clojure.core/set") {
		return "", false
	}
	child := node.Children[1]
	if child == nil || child.Type != reader.NodeList || len(child.Children) == 0 {
		return "", false
	}
	head := child.Children[0]
	if head == nil || head.Type != reader.NodeSymbol || (head.Resolution != nil && head.Resolution.Kind == reader.ResolutionLocal) {
		return "", false
	}
	resolved := rules.ResolvedCall(child)
	if resolved != nil && resolved.Kind != reader.ResolutionUnresolved && resolved.Kind != reader.ResolutionLocal {
		operation, ok := immediatelyRealizedLazyOperations[resolved.CanonicalName]
		return operation, ok
	}
	if head.Resolution == nil || head.Resolution.Kind == reader.ResolutionUnresolved {
		opName := head.Value
		if op, ok := immediatelyRealizedLazyOperations["clojure.core/"+opName]; ok {
			return op, true
		}
	}
	return "", false
}

func (r *UnnecessaryLazinessRule) insideNonEvaluatedContext(context map[string]interface{}) bool {
	return rules.CurrentExecutionContext(context) == rules.ExecutionNonEvaluated ||
		r.IsInside(context, "__non-evaluated__", "comment")
}

func (r *UnnecessaryLazinessRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if r.insideNonEvaluatedContext(context) {
		return nil
	}
	operation, ok := unnecessaryLazyChild(node)
	if !ok {
		return nil
	}
	return &rules.Finding{
		RuleID: r.ID, Filepath: filepath, Location: node.Location, Severity: r.Severity,
		Message: fmt.Sprintf("Lazy operation `%s` is immediately materialized by `vec`. Review whether a direct eager operation or transducer preserves the required type, order, cardinality, effects, chunking, and behavior for unbounded inputs; static analysis does not infer intent.", operation),
	}
}

func init() {
	rules.RegisterRule(&UnnecessaryLazinessRule{Rule: rules.Rule{
		ID: "unnecessary-laziness", Name: "Unnecessary Laziness",
		Description: "Warns when a resolved lazy core operation is immediately materialized by vec; the warning describes a review risk and does not claim an eager replacement is universally equivalent.",
		Severity:    rules.SeverityHint,
	}})
}
