package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type UnnecessaryIntoRule struct {
	rules.Rule
	// Retained for configuration compatibility. Generic type conversions and
	// map rewrites do not currently meet the rule's precision contract.
	CheckTypeTransformations bool `json:"check_type_transformations" yaml:"check_type_transformations"`
	CheckMapMapping          bool `json:"check_map_mapping" yaml:"check_map_mapping"`
	CheckTransducerAPI       bool `json:"check_transducer_api" yaml:"check_transducer_api"`
}

func (r *UnnecessaryIntoRule) Meta() rules.Rule { return r.Rule }

type transducerStepSpec struct {
	canonical string
	args      int
	xformArg  string
}

// These operations have a transducer arity whose arguments are exactly the
// lazy arity minus its final source collection. Operations such as pmap,
// mapv, filterv, for and flatten are deliberately absent.
var fusibleTransducerSteps = []transducerStepSpec{
	{"clojure.core/map", 2, "f"},
	{"clojure.core/filter", 2, "pred"},
	{"clojure.core/remove", 2, "pred"},
	{"clojure.core/keep", 2, "f"},
	{"clojure.core/map-indexed", 2, "f"},
	{"clojure.core/keep-indexed", 2, "f"},
	{"clojure.core/take", 2, "n"},
	{"clojure.core/drop", 2, "n"},
	{"clojure.core/take-while", 2, "pred"},
	{"clojure.core/drop-while", 2, "pred"},
	{"clojure.core/distinct", 1, ""},
	{"clojure.core/dedupe", 1, ""},
}

func exactFusibleTransducerStep(node *reader.RichNode) (transducerStepSpec, bool) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return transducerStepSpec{}, false
	}
	for _, spec := range fusibleTransducerSteps {
		if len(node.Children)-1 == spec.args && resolvesToCoreWithFallback(node, spec.canonical) {
			return spec, true
		}
	}
	return transducerStepSpec{}, false
}

func resolvesToCoreWithFallback(node *reader.RichNode, canonical string) bool {
	if rules.CallResolvesTo(node, canonical) {
		return true
	}
	resolved := rules.ResolvedCall(node)
	if resolved == nil || resolved.Kind != reader.ResolutionUnresolved || len(node.Children) == 0 {
		return false
	}
	return node.Children[0].Value == shortCanonicalName(canonical)
}

func isPlainEmptyVector(node *reader.RichNode) bool {
	return node != nil && node.Type == reader.NodeVector && len(node.Children) == 0 && node.Metadata == nil
}

func shortCanonicalName(canonical string) string {
	for i := len(canonical) - 1; i >= 0; i-- {
		if canonical[i] == '/' {
			return canonical[i+1:]
		}
	}
	return canonical
}

func (r *UnnecessaryIntoRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if !r.CheckTransducerAPI || r.IsInside(context, "__non-evaluated__") ||
		!rules.CallResolvesTo(node, "clojure.core/into") || len(node.Children) != 3 ||
		!isPlainEmptyVector(node.Children[1]) {
		return nil
	}

	step, ok := exactFusibleTransducerStep(node.Children[2])
	if !ok {
		return nil
	}

	operation := shortCanonicalName(step.canonical)
	xformForm := fmt.Sprintf("(%s %s)", operation, step.xformArg)
	if step.xformArg == "" {
		xformForm = fmt.Sprintf("(%s)", operation)
	}
	return &rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"A lazy `%s` result is immediately reduced into a plain vector. Fuse its transducer arity as `(into [] %s source)` to remove only the intermediate lazy sequence while preserving the vector target, order, cardinality, and single evaluation.",
			operation, xformForm,
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func init() {
	rules.RegisterRule(&UnnecessaryIntoRule{
		Rule: rules.Rule{
			ID:          "unnecessary-into",
			Name:        "Unnecessary Into",
			Description: "Detects a lazy intermediate directly consumed by into only when an exact transducer fusion preserves a plain vector target and evaluation semantics.",
			Severity:    rules.SeverityHint,
		},
		CheckTypeTransformations: true,
		CheckMapMapping:          true,
		CheckTransducerAPI:       true,
	})
}
