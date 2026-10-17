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
	case "clojure.core/inc", "clojure.core/dec", "clojure.core/identity",
		"clojure.core/str", "clojure.core/name", "clojure.core/keyword",
		"clojure.core/symbol":
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

// lazinessStandardProducers are core functions whose result is always a core
// collection or lazy sequence, whatever their inputs. Producers that return
// their argument's own type or seq (into, conj, assoc, merge, seq, rest,
// next) are excluded: that type could be a custom reducible.
var lazinessStandardProducers = map[string]bool{
	"range": true, "vec": true, "vector": true, "set": true, "hash-set": true, "hash-map": true,
	"sorted-set": true, "sorted-map": true, "keys": true, "vals": true, "sort": true, "sort-by": true,
	"distinct": true, "concat": true, "map": true, "filter": true, "remove": true, "keep": true,
	"take": true, "drop": true, "reverse": true, "mapv": true, "filterv": true, "partition": true,
	"partition-all": true, "frequencies": true, "group-by": true, "zipmap": true, "interleave": true,
	"interpose": true, "list": true, "re-seq": true, "subvec": true, "mapcat": true, "take-while": true,
	"drop-while": true, "map-indexed": true, "keep-indexed": true, "dedupe": true, "str": true,
}

var lazinessStandardLibraryProducers = map[string]bool{
	"clojure.string/split": true, "clojure.string/split-lines": true,
}

// lazinessSourceHasStandardType proves that a source is a literal or the
// result of a standard producer. For such sources `reduce` (used by mapv and
// filterv) and `seq` (used by map and filter) traverse the same elements in
// the same order, so the rewrite preserves value, effect order, and partial
// effects before an exception (evidence:
// docs/output/audits/clojure-specific/corrections/evidence/lazy_eager_equivalence.clj).
// Only a custom IReduceInit whose reduce disagrees with its seq differs.
func lazinessSourceHasStandardType(node *reader.RichNode, context map[string]interface{}) bool {
	if node == nil {
		return false
	}
	switch node.Type {
	case reader.NodeVector, reader.NodeSet, reader.NodeMap, reader.NodeString, reader.NodeNil:
		return true
	case reader.NodeQuote:
		return len(node.Children) == 1 && node.Children[0] != nil && node.Children[0].Type == reader.NodeList
	case reader.NodeList:
		if len(node.Children) == 0 || node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol {
			return false
		}
		head := node.Children[0]
		if resolution := head.Resolution; resolution != nil {
			switch resolution.Kind {
			case reader.ResolutionClojureCore:
				return lazinessStandardProducers[strings.TrimPrefix(resolution.CanonicalName, "clojure.core/")]
			case reader.ResolutionNamespaceVar:
				return lazinessStandardLibraryProducers[resolution.CanonicalName]
			case reader.ResolutionUnresolved:
				return lazinessStandardProducers[head.Value] && nsReferCoreNameUnambiguously(context, head.Value)
			}
		}
	}
	return false
}

// provenEagerRewrite promotes vec over map/filter when every source has a
// standard type and the eager replacement is available.
func provenEagerRewrite(node *reader.RichNode, context map[string]interface{}) (string, bool) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) != 2 ||
		!rules.CallResolvesTo(node, "clojure.core/vec") {
		return "", false
	}
	child := node.Children[1]
	if child == nil || child.Type != reader.NodeList {
		return "", false
	}
	var eager string
	switch {
	case rules.CallResolvesTo(child, "clojure.core/map") && len(child.Children) >= 3:
		eager = "mapv"
	case rules.CallResolvesTo(child, "clojure.core/filter") && len(child.Children) == 3:
		eager = "filterv"
	default:
		return "", false
	}
	if !isCoreReplacementAvailable(context, eager) || !ancestorsEvaluateNormally(context) {
		return "", false
	}
	for _, source := range child.Children[2:] {
		if !lazinessSourceHasStandardType(source, context) {
			return "", false
		}
	}
	return eager, true
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
	if eager, proven := provenEagerRewrite(node, context); proven {
		if _, pureLiteral := provenEagerMap(node, context); !pureLiteral {
			return &rules.Finding{
				RuleID:     r.ID,
				Filepath:   filepath,
				Location:   node.Location,
				Severity:   r.Severity,
				Confidence: rules.ConfidenceProven,
				Message: fmt.Sprintf(
					"Proven redundant laziness: `%s` over a source of standard collection type is immediately materialized by `vec`; `%s` returns the same vector with the same element order and effect order.",
					operation, eager),
			}
		}
	}
	if mapper, proven := provenEagerMap(node, context); proven {
		return &rules.Finding{
			RuleID:     r.ID,
			Filepath:   filepath,
			Location:   node.Location,
			Severity:   r.Severity,
			Confidence: rules.ConfidenceProven,
			Message: fmt.Sprintf(
				"Proven redundant laziness: `map` with pure `%s` over a finite vector is immediately materialized by `vec`; use `mapv` to preserve the vector result without the intermediate lazy sequence.",
				mapper[strings.LastIndex(mapper, "/")+1:],
			),
		}
	}
	return &rules.Finding{
		RuleID:           r.ID,
		Filepath:         filepath,
		Location:         node.Location,
		Severity:         r.Severity,
		Confidence:       rules.ConfidenceContextual,
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
