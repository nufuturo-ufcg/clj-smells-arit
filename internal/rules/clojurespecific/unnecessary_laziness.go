package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

type UnnecessaryLazinessRule struct{ rules.Rule }

func (r *UnnecessaryLazinessRule) Meta() rules.Rule { return r.Rule }

var immediatelyRealizedLazyOperations = map[string]string{
	"clojure.core/map":          "map",
	"clojure.core/filter":       "filter",
	"clojure.core/keep":         "keep",
	"clojure.core/remove":       "remove",
	"clojure.core/map-indexed":  "map-indexed",
	"clojure.core/keep-indexed": "keep-indexed",
	"clojure.core/take":         "take",
	"clojure.core/drop":         "drop",
	"clojure.core/take-while":   "take-while",
	"clojure.core/drop-while":   "drop-while",
	"clojure.core/distinct":     "distinct",
	"clojure.core/dedupe":       "dedupe",
}

func unnecessaryLazyChild(node *reader.RichNode, context map[string]interface{}) (string, bool) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) != 2 {
		return "", false
	}
	if !rules.CallResolvesTo(node, "clojure.core/vec") {
		return "", false
	}
	child := node.Children[1]
	if child == nil || child.Type != reader.NodeList || len(child.Children) == 0 {
		return "", false
	}
	head := child.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return "", false
	}
	facts, provenShape := rules.ProvenCallFacts(child, context, "")
	if !provenShape || !rules.HasProvenSemanticEvidence(facts) || facts.Resolution == nil || facts.Resolution.Kind == reader.ResolutionLocal ||
		facts.CallMode != semantics.CallModeCollection || facts.Laziness != semantics.Lazy {
		return "", false
	}
	operation, ok := immediatelyRealizedLazyOperations[facts.Resolution.CanonicalName]
	return operation, ok
}

// provenEagerMap identifies the deliberately narrow case where vec(map ...) is
// equivalent to mapv without relying on a caller contract. The source must be
// a finite vector literal and the mapper must be one of the known pure unary
// core functions. Unknown callbacks, bindings, and collection sources remain
// contextual because their effects and protocols are not locally provable.
func provenEagerMap(node *reader.RichNode, context map[string]interface{}) (string, bool) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) != 2 ||
		!rules.CallResolvesTo(node, "clojure.core/vec") {
		return "", false
	}
	child := node.Children[1]
	if child == nil || child.Type != reader.NodeList || len(child.Children) != 3 ||
		!rules.CallResolvesTo(child, "clojure.core/map") {
		return "", false
	}
	mapper := child.Children[1]
	if mapper == nil || mapper.Type != reader.NodeSymbol || mapper.Resolution == nil ||
		mapper.Resolution.Kind != reader.ResolutionClojureCore {
		return "", false
	}
	switch mapper.Resolution.CanonicalName {
	case "clojure.core/inc", "clojure.core/dec", "clojure.core/identity":
		// These core functions are pure and unary, so mapv preserves the
		// observable result for the finite literal source below.
	default:
		return "", false
	}
	source := child.Children[2]
	sourceFacts := rules.FactsForNode(source, context)
	if !rules.HasProvenSemanticEvidence(sourceFacts) || !sourceFacts.Constant ||
		sourceFacts.Type != semantics.TypeVector {
		return "", false
	}
	return mapper.Resolution.CanonicalName, true
}

func (r *UnnecessaryLazinessRule) insideNonEvaluatedContext(context map[string]interface{}) bool {
	return rules.CurrentExecutionContext(context) == rules.ExecutionNonEvaluated ||
		r.IsInside(context, "__non-evaluated__", "comment")
}

func (r *UnnecessaryLazinessRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if r.insideNonEvaluatedContext(context) {
		return nil
	}
	outerFacts := rules.FactsForNode(node, context)
	if !rules.HasProvenCallShape(outerFacts) || outerFacts.CallMode != semantics.CallModeCollection {
		return nil
	}
	operation, ok := unnecessaryLazyChild(node, context)
	if !ok {
		return nil
	}
	if mapper, proven := provenEagerMap(node, context); proven {
		return &rules.Finding{
			RuleID:   r.ID,
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
			Message: fmt.Sprintf(
				"Proven redundant laziness: `map` with pure `%s` over a finite vector is immediately materialized by `vec`; use `mapv` to preserve the vector result without the intermediate lazy sequence.",
				mapper[strings.LastIndex(mapper, "/")+1:],
			),
		}
	}
	return &rules.Finding{
		RuleID: r.ID, Filepath: filepath, Location: node.Location, Severity: r.Severity,
		Contextual:       true,
		ContextualReason: "Equivalence depends on the collection's type contract, cardinality, effects, and protocol.",
		MissingEvidence:  []string{"collection-type", "cardinality", "effects", "protocol"},
		Tags:             []string{"contextual", "contract-dependent"},
		Message:          fmt.Sprintf("Lazy operation `%s` is immediately materialized by `vec`. Review whether a direct eager operation or transducer preserves the required type, order, cardinality, effects, chunking, and behavior for unbounded inputs; static analysis does not infer intent.", operation),
	}
}

func init() {
	rules.RegisterRule(&UnnecessaryLazinessRule{Rule: rules.Rule{
		ID: "unnecessary-laziness", Name: "Unnecessary Laziness",
		Description:           "Warns when a resolved lazy core operation is immediately materialized by vec; the warning describes a review risk and does not claim an eager replacement is universally equivalent.",
		ContextualDescription: "It may be contextual when the collection contract requires lazy evaluation, effect preservation, chunking, or explicit materialization. The finding remains visible with --include-contextual.",
		Severity:              rules.SeverityHint,
	}})
}
