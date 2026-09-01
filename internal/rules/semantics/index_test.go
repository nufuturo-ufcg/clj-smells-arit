package semantics

import (
	"path/filepath"
	"testing"

	"github.com/thlaurentino/arit/internal/reader"
)

func TestProjectIndexEnrichesKnownNamespace(t *testing.T) {
	index := NewProjectIndex()
	index.Namespaces["sample.api"] = "sample_api.clj"

	node := &reader.RichNode{Resolution: &reader.SymbolResolution{
		Kind:      reader.ResolutionNamespaceVar,
		Namespace: "sample.api",
	}}
	index.EnrichResolutions([]*reader.RichNode{node})

	if !node.Resolution.NamespaceKnown {
		t.Fatal("expected project namespace to be marked as known")
	}
}

func TestProjectIndexLeavesUnknownNamespaceUnproven(t *testing.T) {
	index := NewProjectIndex()
	node := &reader.RichNode{Resolution: &reader.SymbolResolution{
		Kind:      reader.ResolutionNamespaceVar,
		Namespace: "external.api",
	}}
	index.EnrichResolutions([]*reader.RichNode{node})

	if node.Resolution.NamespaceKnown {
		t.Fatal("expected external namespace to remain unknown")
	}
}

func TestProjectIndexIndexesMacroDefinitionsAndArities(t *testing.T) {
	index := NewProjectIndex()
	if err := index.IndexFile("../../test/data/multiple_evaluation_in_macros.clj"); err != nil {
		t.Fatalf("index macro fixture: %v", err)
	}

	macro, ok := index.MacroDefinitionOf("multiple-evaluation-in-macros/one-risky-arity")
	if !ok {
		t.Fatal("expected one-risky-arity macro definition")
	}
	if len(macro.Arities) != 2 {
		t.Fatalf("expected two macro arities, got %d", len(macro.Arities))
	}
	if got := macro.Arities[0].Parameters; len(got) != 1 || got[0] != "expr" {
		t.Fatalf("unexpected first arity parameters: %#v", got)
	}
	if got := macro.Arities[1].Parameters; len(got) != 2 || got[0] != "expr" || got[1] != "fallback" {
		t.Fatalf("unexpected second arity parameters: %#v", got)
	}
}

func TestProjectIndexIndexesMacroCallSitesAndConservativeArgumentEvidence(t *testing.T) {
	index := NewProjectIndex()
	provider := filepath.Clean("../../test/data/macro_call_site_provider.clj")
	consumer := filepath.Clean("../../test/data/macro_call_site_consumer.clj")
	if err := index.IndexFile(provider); err != nil {
		t.Fatalf("index macro provider: %v", err)
	}
	if err := index.IndexFile(consumer); err != nil {
		t.Fatalf("index macro consumer: %v", err)
	}
	calls := index.BuildMacroCallSites()
	if index.MacroCallSitesComplete() {
		t.Fatal("caller index must remain incomplete until coverage is explicitly sealed")
	}
	index.MarkMacroCallSitesComplete()
	if !index.MacroCallSitesComplete() {
		t.Fatal("expected caller index to be complete after explicit sealing")
	}
	if len(calls) != 3 {
		t.Fatalf("expected three indexed macro calls, got %d", len(calls))
	}

	resolvedCalls := index.MacroCallSites("macro-call-site.provider/duplicated")
	if len(resolvedCalls) != 3 {
		t.Fatalf("expected three calls to the provider macro, got %d", len(resolvedCalls))
	}
	if resolvedCalls[0].Location.StartLine != 4 || resolvedCalls[0].ArgumentEvidence[0].Effects != EffectMutation || resolvedCalls[0].ArgumentEvidence[0].Evidence != EvidenceProven {
		t.Fatalf("expected explicit swap! call to have proven mutation evidence: %#v", resolvedCalls[0])
	}
	if !resolvedCalls[1].ArgumentEvidence[0].Constant || resolvedCalls[1].ArgumentEvidence[0].Evidence != EvidenceProven {
		t.Fatalf("expected literal argument to have proven constant evidence: %#v", resolvedCalls[1])
	}
	if resolvedCalls[2].ArgumentEvidence[0].Evidence != EvidenceUnknown {
		t.Fatalf("expected unresolved symbol argument to remain unknown: %#v", resolvedCalls[2])
	}
}

func TestProjectIndexResolvesReferredAndRequireMacrosCallSites(t *testing.T) {
	index := NewProjectIndex()
	provider := filepath.Clean("../../test/data/macro_call_site_provider.clj")
	consumer := filepath.Clean("../../test/data/macro_call_site_references.cljc")
	if err := index.IndexFile(provider); err != nil {
		t.Fatalf("index macro provider: %v", err)
	}
	if err := index.IndexFile(consumer); err != nil {
		t.Fatalf("index macro references consumer: %v", err)
	}
	calls := index.BuildMacroCallSites()
	if len(calls) != 2 {
		t.Fatalf("expected two indexed macro calls, got %d: %#v", len(calls), calls)
	}
	for _, call := range calls {
		if call.QualifiedName != "macro-call-site.provider/duplicated" {
			t.Fatalf("unexpected resolved macro call: %#v", call)
		}
	}
}

func TestProjectIndexReportsIncompleteCallerCoverageAfterIndexError(t *testing.T) {
	index := NewProjectIndex()
	missing := filepath.Clean("../../test/data/does-not-exist-for-project-index.clj")
	if err := index.IndexFile(missing); err == nil {
		t.Fatal("expected missing file to produce an index error")
	}
	index.BuildMacroCallSites()
	index.MarkMacroCallSitesComplete()
	if index.MacroCallSitesComplete() {
		t.Fatal("index with an indexing error must not claim complete caller coverage")
	}
	errors := index.IndexErrors()
	if len(errors) != 1 || errors[0] != missing {
		t.Fatalf("unexpected index errors: %#v", errors)
	}
}

func TestProjectIndexDoesNotResolveAmbiguousReferredMacro(t *testing.T) {
	index := NewProjectIndex()
	index.Macros["local.ns/m"] = MacroDefinition{QualifiedName: "local.ns/m"}
	index.Macros["external.ns/m"] = MacroDefinition{QualifiedName: "external.ns/m"}
	if qualified, ok := index.resolveMacroCall("m", "local.ns", nil, map[string]string{"m": "external.ns"}); ok || qualified != "" {
		t.Fatalf("ambiguous referred macro resolved as %q", qualified)
	}
}
