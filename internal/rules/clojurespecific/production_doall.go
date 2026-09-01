package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

type ProductionDoallRule struct {
	rules.Rule
}

func (r *ProductionDoallRule) Meta() rules.Rule { return r.Rule }

var alreadyEagerCollectionProducers = map[string]string{
	"clojure.core/mapv":    "mapv",
	"clojure.core/filterv": "filterv",
	"clojure.core/vec":     "vec",
	"clojure.core/into":    "into",
}

func resolvedEagerCollectionProducer(node *reader.RichNode, context map[string]interface{}) (string, bool) {
	facts, provenShape := rules.ProvenCallFacts(node, context, "")
	if !provenShape || !rules.HasProvenSemanticEvidence(facts) || facts.Laziness != semantics.Eager || facts.Resolution.Kind == reader.ResolutionLocal {
		return "", false
	}
	canonicalName := semantics.CallName(node)
	name, ok := alreadyEagerCollectionProducers[canonicalName]
	return name, ok
}

func eagerProducerExplanation(producer string) string {
	if producer == "into" {
		return "`into` has already consumed and realized its input into the target collection before `doall` runs"
	}
	return fmt.Sprintf("`%s` has already realized every input element into its returned collection before `doall` runs", producer)
}

func (r *ProductionDoallRule) insideNonEvaluatedContext(context map[string]interface{}) bool {
	if rules.CurrentExecutionContext(context) == rules.ExecutionNonEvaluated ||
		r.IsInside(context, "__non-evaluated__", "comment") {
		return true
	}
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil {
			continue
		}
		switch ancestor.Type {
		case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeVarQuote, reader.NodeReaderDiscard:
			return true
		}
		if ancestor.Type == reader.NodeList && len(ancestor.Children) > 0 &&
			ancestor.Children[0].Type == reader.NodeSymbol && ancestor.Children[0].Value == "comment" {
			return true
		}
	}
	return false
}

func (r *ProductionDoallRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if r.insideNonEvaluatedContext(context) ||
		!rules.CallResolvesTo(node, "clojure.core/doall") || len(node.Children) < 1 {
		return nil
	}
	doallFacts := rules.FactsForNode(node, context)
	if !rules.HasProvenCallShape(doallFacts) {
		return nil
	}
	if len(node.Children) != 2 && len(node.Children) != 3 {
		return nil
	}

	if len(node.Children) == 2 {
		if producer, ok := resolvedEagerCollectionProducer(node.Children[1], context); ok {
			return &rules.Finding{
				RuleID: r.ID,
				Message: fmt.Sprintf(
					"Redundant `doall` around `%s`: %s. Remove only the `doall` wrapper; the value, type, order, exceptions, and producer evaluation count are preserved.",
					producer, eagerProducerExplanation(producer),
				),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return &rules.Finding{
		RuleID:           r.ID,
		Message:          "`doall` forces realization and keeps the realized sequence reachable through its return value. Review whether the input is bounded and whether full materialization and retention are required at this lifecycle boundary. Static analysis does not infer intent; if this is deliberate, retain it and document that decision.",
		Filepath:         filepath,
		Location:         node.Location,
		Severity:         r.Severity,
		Contextual:       true,
		ContextualReason: "The need to realize and retain the sequence depends on the producer's lifecycle, cardinality, and effects.",
		MissingEvidence:  []string{"lifecycle", "cardinality", "producer-effects", "ownership"},
		Tags:             []string{"contextual", "review-required"},
	}
}

func init() {
	rules.RegisterRule(&ProductionDoallRule{
		Rule: rules.Rule{
			ID:                    "production-doall",
			Name:                  "Production doall realization review",
			Description:           "Warns on evaluated doall calls so developers can review cardinality, retention, effects, and lifecycle boundaries; identifies already-eager vector producers as proven redundancy.",
			ContextualDescription: "It may be contextual when doall is used to close a resource, await work, anticipate effects, or materialize a response. The finding remains visible with --include-contextual.",
			Severity:              rules.SeverityWarning,
		},
	})
}
