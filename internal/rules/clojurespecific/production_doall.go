package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type ProductionDoallRule struct {
	rules.Rule
}

func (r *ProductionDoallRule) Meta() rules.Rule { return r.Rule }

var alreadyEagerVectorProducers = map[string]string{
	"clojure.core/mapv":    "mapv",
	"clojure.core/filterv": "filterv",
}

func resolvedEagerVectorProducer(node *reader.RichNode) (string, bool) {
	resolved := rules.ResolvedCall(node)
	if resolved == nil || resolved.Kind == reader.ResolutionUnresolved || resolved.Kind == reader.ResolutionLocal {
		return "", false
	}
	name, ok := alreadyEagerVectorProducers[resolved.CanonicalName]
	return name, ok
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

	if len(node.Children) == 2 {
		if producer, ok := resolvedEagerVectorProducer(node.Children[1]); ok {
			return &rules.Finding{
				RuleID: r.ID,
				Message: fmt.Sprintf(
					"Redundant `doall` around `%s`: `%s` has already realized every input element into a persistent vector before `doall` runs. Remove only the `doall` wrapper; the value, type, order, exceptions, and producer evaluation count are preserved.",
					producer, producer,
				),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return &rules.Finding{
		RuleID:   r.ID,
		Message:  "`doall` forces realization and keeps the realized sequence reachable through its return value. Review whether the input is bounded and whether full materialization and retention are required at this lifecycle boundary. Static analysis does not infer intent; if this is deliberate, retain it and document that decision.",
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func init() {
	rules.RegisterRule(&ProductionDoallRule{
		Rule: rules.Rule{
			ID:          "production-doall",
			Name:        "Production doall realization review",
			Description: "Warns on evaluated doall calls so developers can review cardinality, retention, effects, and lifecycle boundaries; identifies already-eager vector producers as proven redundancy.",
			Severity:    rules.SeverityWarning,
		},
	})
}
