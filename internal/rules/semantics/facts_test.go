package semantics

import (
	"testing"

	"github.com/thlaurentino/arit/internal/reader"
)

type testSemanticBinding struct {
	typeHint     string
	inferredType string
	bindingValue *reader.RichNode
}

func (b *testSemanticBinding) SemanticTypeHint() string               { return b.typeHint }
func (b *testSemanticBinding) SemanticInferredType() string           { return b.inferredType }
func (b *testSemanticBinding) SemanticBindingValue() *reader.RichNode { return b.bindingValue }

func TestForNodeClassifiesEagerProducer(t *testing.T) {
	node := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "mapv", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionClojureCore, CanonicalName: "clojure.core/mapv",
			}},
			{Type: reader.NodeSymbol, Value: "f"},
			{Type: reader.NodeVector},
		},
	}

	facts := ForNode(node, map[string]interface{}{"executionContext": "load"})
	if facts.Laziness != Eager {
		t.Fatalf("expected eager producer, got %q", facts.Laziness)
	}
	if facts.SemanticEvidence != EvidenceProven {
		t.Fatalf("expected proven semantic evidence, got %q", facts.SemanticEvidence)
	}
}

func TestForNodeCacheSeparatesContextSensitiveFacts(t *testing.T) {
	node := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "map", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionClojureCore, CanonicalName: "clojure.core/map",
			}},
			{Type: reader.NodeSymbol, Value: "f"},
			{Type: reader.NodeVector},
		},
	}
	cache := NewFactsCache()
	loadContext := map[string]interface{}{
		"executionContext":     "load",
		"semantic-facts-cache": cache,
	}
	deferredContext := map[string]interface{}{
		"executionContext":     "deferred",
		"semantic-facts-cache": cache,
	}

	load := ForNode(node, loadContext)
	deferred := ForNode(node, deferredContext)
	if load.Execution != ExecutionAtLoad || deferred.Execution != ExecutionDeferred {
		t.Fatalf("cache reused execution-sensitive facts: load=%q deferred=%q", load.Execution, deferred.Execution)
	}
	if len(cache.entries) != 2 {
		t.Fatalf("expected one cached entry per execution phase, got %d", len(cache.entries))
	}
	if repeated := ForNode(node, loadContext); repeated.Execution != ExecutionAtLoad {
		t.Fatalf("cache returned incorrect repeated facts: %q", repeated.Execution)
	}
}

func TestForNodeMarksUnknownCallEffect(t *testing.T) {
	node := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "external-call"},
		},
	}

	facts := ForNode(node, nil)
	if !facts.Effects.Has(EffectUnknown) {
		t.Fatal("expected unresolved calls to carry unknown-effect evidence")
	}
	if facts.SemanticEvidence != EvidenceUnknown {
		t.Fatalf("expected unknown semantic evidence, got %q", facts.SemanticEvidence)
	}
}

func TestForNodeDoesNotTreatNestedSymbolCollectionAsConstant(t *testing.T) {
	nested := &reader.RichNode{
		Type: reader.NodeVector,
		Children: []*reader.RichNode{
			{Type: reader.NodeVector, Children: []*reader.RichNode{{Type: reader.NodeSymbol, Value: "value"}}},
		},
	}
	facts := ForNode(nested, nil)
	if facts.Literal || facts.Constant {
		t.Fatal("expected a collection containing a symbol to remain non-constant")
	}

	literal := ForNode(&reader.RichNode{
		Type:     reader.NodeVector,
		Children: []*reader.RichNode{{Type: reader.NodeVector, Children: []*reader.RichNode{{Type: reader.NodeNumber, Value: "1"}}}},
	}, nil)
	if !literal.Literal || !literal.Constant || literal.EvidenceSource != SourceSyntaxLiteral {
		t.Fatalf("expected nested literal collection facts, got %#v", literal)
	}
}

func TestForNodeCachesNestedLiteralCollectionsBottomUp(t *testing.T) {
	inner := &reader.RichNode{
		Type:     reader.NodeVector,
		Children: []*reader.RichNode{{Type: reader.NodeNumber, Value: "1"}},
	}
	outer := &reader.RichNode{
		Type:     reader.NodeVector,
		Children: []*reader.RichNode{inner},
	}
	cache := NewFactsCache()
	context := map[string]interface{}{"semantic-facts-cache": cache}

	facts := ForNode(outer, context)
	if !facts.Literal || !facts.Constant || facts.ConstantValue != outer.Value {
		t.Fatalf("expected cached outer literal facts, got %#v", facts)
	}
	if len(cache.literalEntries) != 2 {
		t.Fatalf("expected outer and inner collections in bottom-up cache, got %d", len(cache.literalEntries))
	}

	if repeated := ForNode(inner, context); !repeated.Literal || !repeated.Constant {
		t.Fatalf("expected repeated inner lookup to remain literal, got %#v", repeated)
	}
}

func TestForNodeReusesProvenLocalBindingFacts(t *testing.T) {
	bindingValue := &reader.RichNode{Type: reader.NodeVector, Children: []*reader.RichNode{
		{Type: reader.NodeNumber, Value: "1"},
	}}
	node := &reader.RichNode{
		Type: reader.NodeSymbol, Value: "items",
		Resolution: &reader.SymbolResolution{Kind: reader.ResolutionLocal},
		SymbolRef:  &testSemanticBinding{bindingValue: bindingValue},
	}
	facts := ForNode(node, map[string]interface{}{"semantic-options": map[string]bool{"type-inference": true}})
	if !facts.Literal || !facts.Constant || facts.Type != TypeVector || facts.Nilability != NonNil {
		t.Fatalf("expected direct local literal binding facts, got %#v", facts)
	}
	if facts.SemanticEvidence != EvidenceProven || facts.EvidenceSource != SourceLocalBinding {
		t.Fatalf("expected local-binding provenance, got %#v", facts)
	}
	if facts.Effects != EffectNone {
		t.Fatalf("reading a local binding must not inherit initializer effects, got %v", facts.Effects)
	}
}

func TestForNodeDoesNotPromoteUnknownLocalBinding(t *testing.T) {
	value := &reader.RichNode{Type: reader.NodeList, Children: []*reader.RichNode{{Type: reader.NodeSymbol, Value: "external"}}}
	node := &reader.RichNode{
		Type: reader.NodeSymbol, Value: "value",
		Resolution: &reader.SymbolResolution{Kind: reader.ResolutionLocal},
		SymbolRef:  &testSemanticBinding{bindingValue: value},
	}
	facts := ForNode(node, nil)
	if facts.Constant || facts.Literal || facts.Type != TypeSymbol {
		t.Fatalf("unknown local initializer must not be promoted, got %#v", facts)
	}
	if facts.SemanticEvidence != EvidenceProven || facts.EvidenceSource != SourceLexicalResolution {
		t.Fatalf("expected only lexical provenance, got %#v", facts)
	}
}

func TestForNodeDoesNotInferEffectsFromUnresolvedQualifiedName(t *testing.T) {
	node := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "example/add", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionUnresolved, CanonicalName: "example/add",
			}},
		},
	}
	facts := ForNode(node, nil)
	if facts.Effects.Has(EffectMutation) {
		t.Fatal("unresolved qualified call must not be classified as mutation by its suffix")
	}
	if !facts.Effects.Has(EffectUnknown) || facts.SemanticEvidence != EvidenceUnknown {
		t.Fatalf("expected unknown effects and evidence, got %#v", facts)
	}
}

func TestForNodeRecordsCanonicalArityAndRejectsInvalidCall(t *testing.T) {
	valid := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "map", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionClojureCore, CanonicalName: "clojure.core/map",
			}},
			{Type: reader.NodeSymbol, Value: "f"},
			{Type: reader.NodeVector},
			{Type: reader.NodeVector},
		},
	}
	facts := ForNode(valid, nil)
	if !facts.ArityKnown || !facts.ArityValid || facts.ArgumentCount != 3 {
		t.Fatalf("expected valid canonical arity, got %#v", facts)
	}

	invalid := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "map", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionClojureCore, CanonicalName: "clojure.core/map",
			}},
		},
	}
	invalidFacts := ForNode(invalid, nil)
	if !invalidFacts.ArityKnown || invalidFacts.ArityValid || !invalidFacts.Effects.Has(EffectUnknown) || invalidFacts.SemanticEvidence != EvidenceUnknown {
		t.Fatalf("expected invalid canonical arity to remain unknown, got %#v", invalidFacts)
	}
}

func TestCanonicalArityValidSupportsTransformedCalls(t *testing.T) {
	if known, valid := CanonicalArityValid("clojure.core/assoc", 3); !known || !valid {
		t.Fatalf("expected assoc with three arguments to be valid, got known=%v valid=%v", known, valid)
	}
	if known, valid := CanonicalArityValid("clojure.core/assoc", 4); !known || valid {
		t.Fatalf("expected assoc with four arguments to be invalid, got known=%v valid=%v", known, valid)
	}
	if known, valid := CanonicalArityValid("sample/unknown", 2); known || valid {
		t.Fatalf("expected unknown operation to remain unknown, got known=%v valid=%v", known, valid)
	}

	tests := []struct {
		name      string
		arguments int
		valid     bool
	}{
		{"clojure.core/atom", 1, true},
		{"clojure.core/atom", 2, false},
		{"clojure.core/atom", 3, true},
		{"clojure.core/volatile!", 1, true},
		{"clojure.core/volatile!", 2, false},
		{"clojure.core.async/put!", 2, true},
		{"clojure.core.async/put!", 1, false},
		{"clojure.core.async/>!", 2, true},
		{"clojure.core.async/>!", 3, false},
		{"clojure.core.async/<!", 1, true},
		{"clojure.core.async/<!", 0, false},
		{"clojure.core/defmulti", 2, true},
		{"clojure.core/defmethod", 4, true},
		{"clojure.core/defmethod", 3, false},
		{"clojure.core/slurp", 2, true},
		{"clojure.core/slurp", 3, false},
		{"clojure.core/spit", 2, true},
		{"clojure.core/await", 1, true},
		{"Thread/sleep", 1, true},
		{"Thread/sleep", 2, false},
	}
	for _, test := range tests {
		known, valid := CanonicalArityValid(test.name, test.arguments)
		if !known || valid != test.valid {
			t.Errorf("CanonicalArityValid(%q, %d) = known=%v valid=%v, want known=true valid=%v", test.name, test.arguments, known, valid, test.valid)
		}
	}
}

func TestForNodeDistinguishesCollectionAndTransducerModes(t *testing.T) {
	makeCall := func(args ...*reader.RichNode) *reader.RichNode {
		return &reader.RichNode{
			Type: reader.NodeList,
			Children: append([]*reader.RichNode{{Type: reader.NodeSymbol, Value: "map", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionClojureCore, CanonicalName: "clojure.core/map",
			}}}, args...),
		}
	}

	transducer := ForNode(makeCall(&reader.RichNode{Type: reader.NodeSymbol, Value: "inc"}), nil)
	if !transducer.ArityKnown || !transducer.ArityValid || transducer.CallMode != CallModeTransducer || transducer.Type != TypeFunction || transducer.Laziness != LazinessUnknown || transducer.DataArgumentIndex != -1 {
		t.Fatalf("expected map transducer mode without data position, got %#v", transducer)
	}

	collection := ForNode(makeCall(
		&reader.RichNode{Type: reader.NodeSymbol, Value: "inc"},
		&reader.RichNode{Type: reader.NodeVector},
	), nil)
	if collection.CallMode != CallModeCollection || collection.Laziness != Lazy || collection.DataPosition != DataPositionLast || collection.DataArgumentIndex != 1 {
		t.Fatalf("expected map collection mode with last data argument, got %#v", collection)
	}
}

func TestForNodeLeavesAmbiguousMultiArgumentPositionUnknown(t *testing.T) {
	node := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "into", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionClojureCore, CanonicalName: "clojure.core/into",
			}},
			{Type: reader.NodeVector},
			{Type: reader.NodeVector},
		},
	}
	facts := ForNode(node, nil)
	if !facts.ArityKnown || !facts.ArityValid || facts.DataPosition != DataPositionUnknown || facts.DataArgumentIndex != -1 {
		t.Fatalf("expected into's multi-argument data position to remain unknown, got %#v", facts)
	}
}

func TestForNodeSeparatesProjectResolutionFromFunctionContract(t *testing.T) {
	index := NewProjectIndex()
	index.Namespaces["sample.api"] = "sample_api.clj"
	node := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "api/run", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionNamespaceVar, CanonicalName: "sample.api/run", Namespace: "sample.api",
			}},
		},
	}
	facts := ForNode(node, map[string]interface{}{"project-index": index})
	if facts.ResolutionSource != SourceProjectIndex {
		t.Fatalf("expected project-index resolution source, got %#v", facts)
	}
	if facts.EvidenceSource != SourceUnknown || facts.SemanticEvidence != EvidenceUnknown {
		t.Fatalf("namespace presence must not prove an external function contract, got %#v", facts)
	}

	withoutIndex := ForNode(node, nil)
	if withoutIndex.ResolutionSource != SourceCanonicalResolution {
		t.Fatalf("expected canonical resolution without project index, got %#v", withoutIndex)
	}
}

func TestForNodeUsesOnlyExplicitValidFunctionContracts(t *testing.T) {
	makeCall := func(name string) *reader.RichNode {
		return &reader.RichNode{
			Type: reader.NodeList,
			Children: []*reader.RichNode{
				{Type: reader.NodeSymbol, Value: name, Resolution: &reader.SymbolResolution{
					Kind: reader.ResolutionNamespaceVar, CanonicalName: name,
				}},
				{Type: reader.NodeString, Value: "resource"},
			},
		}
	}
	context := map[string]interface{}{
		"semantic-contracts": map[string]map[string]interface{}{
			"sample/read": {
				"effects":             []interface{}{"io"},
				"returns-type":        "string",
				"laziness":            "eager",
				"arity":               1,
				"data-position":       "single",
				"data-argument-index": 0,
			},
			"sample/invalid": {
				"effects": []interface{}{"not-an-effect"},
			},
			"sample/partial": {
				"returns-type": "string",
			},
		},
	}

	facts := ForNode(makeCall("sample/read"), context)
	if facts.Effects != EffectIO || facts.Type != TypeString || facts.Laziness != Eager ||
		!facts.ArityKnown || !facts.ArityValid || facts.DataPosition != DataPositionSingle ||
		facts.DataArgumentIndex != 0 || facts.EvidenceSource != SourceConfiguredContract ||
		facts.SemanticEvidence != EvidenceProven {
		t.Fatalf("expected explicit contract facts, got %#v", facts)
	}

	invalid := ForNode(makeCall("sample/invalid"), context)
	if invalid.EvidenceSource == SourceConfiguredContract || invalid.SemanticEvidence != EvidenceUnknown {
		t.Fatalf("invalid contract must not promote evidence, got %#v", invalid)
	}

	partial := makeCall("sample/partial")
	partial.Children[0].Resolution = nil
	partialFacts := ForNode(partial, context)
	if partialFacts.EvidenceSource == SourceConfiguredContract || !partialFacts.Effects.Has(EffectUnknown) ||
		partialFacts.SemanticEvidence != EvidenceUnknown {
		t.Fatalf("partial contract must not hide unknown call effects, got %#v", partialFacts)
	}
}

func TestForNodeDoesNotOverrideCanonicalOrLocalFactsWithConfiguredContracts(t *testing.T) {
	context := map[string]interface{}{
		"semantic-contracts": map[string]map[string]interface{}{
			"clojure.core/slurp": {"effects": []interface{}{"none"}},
			"local/read":         {"effects": []interface{}{"none"}},
		},
	}
	canonical := &reader.RichNode{Type: reader.NodeList, Children: []*reader.RichNode{
		{Type: reader.NodeSymbol, Value: "slurp", Resolution: &reader.SymbolResolution{
			Kind: reader.ResolutionClojureCore, CanonicalName: "clojure.core/slurp",
		}},
		{Type: reader.NodeString, Value: "resource"},
	}}
	if facts := ForNode(canonical, context); !facts.Effects.Has(EffectIO) || facts.EvidenceSource == SourceConfiguredContract {
		t.Fatalf("canonical semantics must not be overridden by config, got %#v", facts)
	}

	local := &reader.RichNode{Type: reader.NodeList, Children: []*reader.RichNode{
		{Type: reader.NodeSymbol, Value: "read", Resolution: &reader.SymbolResolution{
			Kind: reader.ResolutionLocal, CanonicalName: "local/read",
		}},
	}}
	if facts := ForNode(local, context); facts.EvidenceSource == SourceConfiguredContract {
		t.Fatalf("local semantics must not be overridden by config, got %#v", facts)
	}
}

func TestBuildFunctionSummariesTracksEffectsAndReturnShape(t *testing.T) {
	mapNode := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "map", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionClojureCore, CanonicalName: "clojure.core/map",
			}},
		},
	}
	root := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "defn"},
			{Type: reader.NodeSymbol, Value: "lazy-values"},
			{Type: reader.NodeVector},
			mapNode,
		},
	}

	summaries := BuildFunctionSummaries([]*reader.RichNode{root}, "sample")
	summary, ok := summaries["sample/lazy-values"]
	if !ok {
		t.Fatal("expected qualified function summary")
	}
	if !summary.ReturnsLazy {
		t.Fatal("expected lazy return summary")
	}
	if summary.ReturnsType != TypeList {
		t.Fatalf("expected list return type, got %q", summary.ReturnsType)
	}
}

func TestBuildFunctionSummariesExcludesDeferredEffects(t *testing.T) {
	blocking := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "clojure.core.async/<!!", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionNamespaceVar, CanonicalName: "clojure.core.async/<!!",
			}},
		},
	}
	future := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "future", Resolution: &reader.SymbolResolution{
				Kind: reader.ResolutionClojureCore, CanonicalName: "clojure.core/future",
			}},
			blocking,
		},
	}
	root := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "defn"},
			{Type: reader.NodeSymbol, Value: "deferred"},
			{Type: reader.NodeVector},
			future,
		},
	}

	summary := BuildFunctionSummaries([]*reader.RichNode{root}, "sample")["sample/deferred"]
	if !summary.Effects.Has(EffectBlocking) {
		t.Fatal("expected broad effects to retain the deferred blocking operation")
	}
	if summary.EagerEffects.Has(EffectBlocking) {
		t.Fatal("did not expect deferred blocking operation in eager effects")
	}
}

func TestBuildFunctionSummariesExcludesNestedDeferredBodies(t *testing.T) {
	makeCall := func(name string, canonical string) *reader.RichNode {
		return &reader.RichNode{
			Type: reader.NodeList,
			Children: []*reader.RichNode{
				{Type: reader.NodeSymbol, Value: name, Resolution: &reader.SymbolResolution{
					Kind: reader.ResolutionNamespaceVar, CanonicalName: canonical,
				}},
			},
		}
	}
	makeFunction := func(name string, body *reader.RichNode) *reader.RichNode {
		return &reader.RichNode{
			Type: reader.NodeList,
			Children: []*reader.RichNode{
				{Type: reader.NodeSymbol, Value: "defn"},
				{Type: reader.NodeSymbol, Value: name},
				{Type: reader.NodeVector},
				body,
			},
		}
	}
	makeDeferred := func(head string, body *reader.RichNode) *reader.RichNode {
		return &reader.RichNode{
			Type: reader.NodeList,
			Children: []*reader.RichNode{
				{Type: reader.NodeSymbol, Value: head},
				body,
			},
		}
	}

	deferredForms := []*reader.RichNode{
		{
			Type:     reader.NodeFnLiteral,
			Children: []*reader.RichNode{makeCall("clojure.core/slurp", "clojure.core/slurp")},
		},
		makeDeferred("fn", makeCall("clojure.core/slurp", "clojure.core/slurp")),
		makeDeferred("defn", makeCall("clojure.core/slurp", "clojure.core/slurp")),
		makeDeferred("delay", makeCall("clojure.core/slurp", "clojure.core/slurp")),
		makeDeferred("future", makeCall("clojure.core/slurp", "clojure.core/slurp")),
	}

	for index, deferred := range deferredForms {
		name := "deferred-" + string(rune('a'+index))
		summary := BuildFunctionSummaries([]*reader.RichNode{makeFunction(name, deferred)}, "sample")["sample/"+name]
		if !summary.Effects.Has(EffectIO) {
			t.Fatalf("case %d: expected broad effects to retain nested IO", index)
		}
		if summary.EagerEffects.Has(EffectIO) {
			t.Fatalf("case %d: did not expect nested deferred IO in eager effects", index)
		}
		if summary.EagerEvidence != EvidenceProven {
			t.Fatalf("case %d: skipping deferred code must not make eager evidence unknown, got %q", index, summary.EagerEvidence)
		}
	}

	direct := BuildFunctionSummaries([]*reader.RichNode{
		makeFunction("direct", makeCall("clojure.core/slurp", "clojure.core/slurp")),
	}, "sample")["sample/direct"]
	if !direct.EagerEffects.Has(EffectIO) {
		t.Fatal("expected an immediate IO call to remain in eager effects")
	}
}
