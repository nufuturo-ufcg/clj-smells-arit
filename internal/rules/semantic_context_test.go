package rules

import (
	"testing"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

func TestProvenSemanticFactsRequiresExplicitProof(t *testing.T) {
	proven := &semantics.Facts{SemanticEvidence: semantics.EvidenceProven}
	unknown := &semantics.Facts{SemanticEvidence: semantics.EvidenceUnknown}

	if !HasProvenSemanticEvidence(proven) {
		t.Fatal("expected proven facts to pass the semantic gate")
	}
	if HasProvenSemanticEvidence(unknown) || HasProvenSemanticEvidence(nil) {
		t.Fatal("unknown or missing facts must not pass the semantic gate")
	}

	context := map[string]interface{}{semanticFactsContextKey: proven}
	if got := ProvenSemanticFacts(context); got != proven {
		t.Fatal("expected the gate to return the original proven facts")
	}
	context[semanticFactsContextKey] = unknown
	if got := ProvenSemanticFacts(context); got != nil {
		t.Fatal("expected unknown facts to be rejected by the promotion gate")
	}
}

func TestHasProvenTypeFactAcceptsTypedCollectionLiteral(t *testing.T) {
	facts := &semantics.Facts{
		Node:             &reader.RichNode{Type: reader.NodeVector},
		Type:             semantics.TypeVector,
		SemanticEvidence: semantics.EvidenceUnknown,
	}
	if !HasProvenTypeFact(facts) {
		t.Fatal("a vector literal must prove its outer collection type")
	}
	if HasProvenTypeFact(&semantics.Facts{Type: semantics.TypeVector}) {
		t.Fatal("an inferred collection type without evidence must not pass")
	}
}

func TestHasProvenCallShapeRequiresResolutionAndValidArity(t *testing.T) {
	valid := &semantics.Facts{ResolutionKnown: true, ArityKnown: true, ArityValid: true}
	if !HasProvenCallShape(valid) {
		t.Fatal("expected resolved valid call shape to pass")
	}
	for _, facts := range []*semantics.Facts{
		{ResolutionKnown: false, ArityKnown: true, ArityValid: true},
		{ResolutionKnown: true, ArityKnown: false, ArityValid: true},
		{ResolutionKnown: true, ArityKnown: true, ArityValid: false},
		nil,
	} {
		if HasProvenCallShape(facts) {
			t.Fatalf("invalid call shape passed the gate: %#v", facts)
		}
	}
}

func TestFactsForNodeUsesCurrentFactsOnlyForMatchingNode(t *testing.T) {
	currentNode := &reader.RichNode{Type: reader.NodeNumber, Value: "1"}
	otherNode := &reader.RichNode{Type: reader.NodeNumber, Value: "2"}
	current := &semantics.Facts{Node: currentNode, Type: semantics.TypeNumber}
	context := map[string]interface{}{semanticFactsContextKey: current}

	if got := FactsForNode(currentNode, context); got != current {
		t.Fatal("expected the current node facts to be reused")
	}
	if got := FactsForNode(otherNode, context); got == current || got.Node != otherNode {
		t.Fatal("expected a different node to use independently computed facts")
	}
}

func TestProvenCallFactsRequiresCanonicalResolutionAndValidArity(t *testing.T) {
	node := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{{
			Type:  reader.NodeSymbol,
			Value: "mapv",
			Resolution: &reader.SymbolResolution{
				Kind:          reader.ResolutionClojureCore,
				CanonicalName: "clojure.core/mapv",
			},
		}, {Type: reader.NodeSymbol, Value: "f"}, {Type: reader.NodeVector}},
	}
	context := map[string]interface{}{"semantic-facts": &semantics.Facts{
		Node:            node,
		Resolution:      node.Children[0].Resolution,
		ResolutionKnown: true,
		ArityKnown:      true,
		ArityValid:      true,
	}}
	if facts, ok := ProvenCallFacts(node, context, "clojure.core/mapv"); !ok || facts.Node != node {
		t.Fatal("expected the canonical resolved call to pass the shared gate")
	}
	if _, ok := ProvenCallFacts(node, context, "clojure.core/map"); ok {
		t.Fatal("a different canonical call must not pass the shared gate")
	}
}
