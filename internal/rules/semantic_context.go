package rules

import "github.com/thlaurentino/arit/internal/rules/semantics"

import "github.com/thlaurentino/arit/internal/reader"

const semanticFactsContextKey = "semantic-facts"

func SemanticFacts(context map[string]interface{}) *semantics.Facts {
	if context == nil {
		return nil
	}
	facts, _ := context[semanticFactsContextKey].(*semantics.Facts)
	return facts
}

// FactsForNode returns the shared semantic facts for node. The analyzer puts
// the current node's facts in the context for the common path; child queries
// must go through this helper so they use the same cache and configuration.
func FactsForNode(node *reader.RichNode, context map[string]interface{}) *semantics.Facts {
	if current := SemanticFacts(context); current != nil && current.Node == node {
		return current
	}
	facts := semantics.ForNode(node, context)
	return &facts
}

// ProvenCallFacts centralizes the minimum proof required before a rule can
// make a semantic claim about a canonical call: known resolution and valid
// arity. Callers may additionally require HasProvenSemanticEvidence when the
// claim depends on effects, type, laziness, or another inferred property.
func ProvenCallFacts(node *reader.RichNode, context map[string]interface{}, canonical string) (*semantics.Facts, bool) {
	facts := FactsForNode(node, context)
	if !HasProvenCallShape(facts) || facts.Resolution == nil {
		return facts, false
	}
	if canonical != "" && facts.Resolution.CanonicalName != canonical {
		return facts, false
	}
	return facts, true
}

// HasProvenSemanticEvidence is the single promotion gate for rules that need
// semantic facts rather than a syntactic or contextual hint. Callers should
// use it before emitting a non-contextual semantic finding.
func HasProvenSemanticEvidence(facts *semantics.Facts) bool {
	return facts != nil && facts.SemanticEvidence == semantics.EvidenceProven
}

// HasProvenCallShape verifies the minimum semantic proof needed by rules that
// reason about a canonical operation's syntax: resolved identity and valid
// arity. It is intentionally weaker than HasProvenSemanticEvidence because
// pure operators such as `=` have no effects/type summary to populate the
// broader evidence field.
func HasProvenCallShape(facts *semantics.Facts) bool {
	return facts != nil && facts.ResolutionKnown && facts.ArityKnown && facts.ArityValid
}

// ProvenSemanticFacts returns the current facts only when their semantic
// evidence is proven. It keeps promotion decisions explicit and prevents a
// missing facts context from being treated as proof.
func ProvenSemanticFacts(context map[string]interface{}) *semantics.Facts {
	facts := SemanticFacts(context)
	if !HasProvenSemanticEvidence(facts) {
		return nil
	}
	return facts
}

// HasProvenTypeFact is the field-specific companion to the general promotion
// gate. A collection literal proves its own outer type even when one of its
// elements is dynamic; other inferred types still require proven evidence.
func HasProvenTypeFact(facts *semantics.Facts) bool {
	if facts == nil {
		return false
	}
	if HasProvenSemanticEvidence(facts) {
		return true
	}
	if facts.Node == nil {
		return false
	}
	switch facts.Node.Type {
	case reader.NodeString, reader.NodeVector, reader.NodeMap, reader.NodeSet:
		return true
	default:
		return false
	}
}

func SetSemanticFacts(context map[string]interface{}, facts *semantics.Facts) {
	if context != nil {
		context[semanticFactsContextKey] = facts
	}
}

func FunctionSummaries(context map[string]interface{}) map[string]semantics.FunctionSummary {
	if context == nil {
		return nil
	}
	summaries, _ := context["function-summaries"].(map[string]semantics.FunctionSummary)
	return summaries
}

func ProjectSemanticIndex(context map[string]interface{}) *semantics.ProjectIndex {
	if context == nil {
		return nil
	}
	index, _ := context["project-index"].(*semantics.ProjectIndex)
	return index
}

func SemanticOption(context map[string]interface{}, name string) bool {
	if context == nil {
		return false
	}
	options, _ := context["semantic-options"].(map[string]bool)
	return options[name]
}
