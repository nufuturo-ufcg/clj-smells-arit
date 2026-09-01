package analyzer

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

func resolutionTestFile(t *testing.T, name string) AnalysisResult {
	t.Helper()

	path, err := filepath.Abs(filepath.Join("../test/data", name))
	if err != nil {
		t.Fatal(err)
	}
	result, err := AnalyzeFile(path, &config.Config{
		EnabledRules:  map[string]bool{},
		EnabledGroups: map[string]bool{},
		RuleConfig:    map[string]config.RuleSettings{},
	})
	if err != nil {
		t.Fatalf("analyze %s: %v", name, err)
	}
	return result
}

func TestAnalysisTimingIsDiagnosticOnly(t *testing.T) {
	previous := EnableTiming
	t.Cleanup(func() { EnableTiming = previous })

	EnableTiming = true
	timed := resolutionTestFile(t, "resolution_semantics.clj")
	if timed.Timings.Parse <= 0 || timed.Timings.RuleTraversal <= 0 {
		t.Fatalf("expected timing instrumentation to populate parse and rule traversal: %#v", timed.Timings)
	}

	EnableTiming = false
	untimed := resolutionTestFile(t, "resolution_semantics.clj")
	if !reflect.DeepEqual(timed.Findings, untimed.Findings) {
		t.Fatalf("timing instrumentation changed findings:\n timed=%#v\nuntimed=%#v", timed.Findings, untimed.Findings)
	}
	if untimed.Timings != (PhaseTimings{}) {
		t.Fatalf("timings should remain empty when instrumentation is disabled: %#v", untimed.Timings)
	}
}

func TestShadowedCoreSymbolsAreCachedPerScope(t *testing.T) {
	scope := NewScope(nil)
	scope.Define(&SymbolInfo{Name: "count", Type: TypeVariable})
	cache := make(map[*Scope]map[string]bool)

	first := cachedShadowedCoreSymbols(cache, scope)
	if !first["count"] {
		t.Fatalf("expected count to be recognized as shadowed: %#v", first)
	}
	first["__cache_sentinel__"] = true
	second := cachedShadowedCoreSymbols(cache, scope)
	if !second["__cache_sentinel__"] {
		t.Fatal("expected repeated lookup to reuse the per-scope cached map")
	}
}

func TestProjectIndexAnalysisReusesIndexedRichTree(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("../test/data", "improper_emptiness_check.clj"))
	if err != nil {
		t.Fatal(err)
	}
	projectIndex := semantics.NewProjectIndex()
	if err := projectIndex.IndexFile(path); err != nil {
		t.Fatalf("index %s: %v", path, err)
	}
	indexedRoots, indexedComments, ok := projectIndex.CachedFile(path)
	if !ok || len(indexedRoots) == 0 {
		t.Fatal("expected indexed AST to be available for reuse")
	}

	previous := EnableTiming
	t.Cleanup(func() { EnableTiming = previous })
	EnableTiming = true
	result, err := NewAnalyzer(&config.Config{
		EnabledRules:  map[string]bool{},
		EnabledGroups: map[string]bool{},
		RuleConfig:    map[string]config.RuleSettings{},
	}).AnalyzeFileWithProjectIndex(path, projectIndex)
	if err != nil {
		t.Fatalf("analyze %s: %v", path, err)
	}
	if result.Timings.Parse != 0 || result.Timings.BuildRichTree != 0 {
		t.Fatalf("expected indexed AST reuse to skip parse/build, got %#v", result.Timings)
	}
	if len(result.RichRoots) != len(indexedRoots) || result.RichRoots[0] != indexedRoots[0] {
		t.Fatal("analysis did not reuse the indexed rich AST")
	}
	if indexedComments == nil {
		t.Fatal("expected indexed comments to be preserved")
	}
}

func resolutionSymbolAt(t *testing.T, roots []*reader.RichNode, line int, value string) *reader.RichNode {
	t.Helper()
	var found *reader.RichNode
	var visit func(*reader.RichNode)
	visit = func(node *reader.RichNode) {
		if node == nil || found != nil {
			return
		}
		if node.Type == reader.NodeSymbol && node.Value == value && node.Location != nil && node.Location.StartLine == line {
			found = node
			return
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	for _, root := range roots {
		visit(root)
	}
	if found == nil {
		t.Fatalf("symbol %q at line %d not found", value, line)
	}
	return found
}

func TestResolutionSemanticsAliasesReferAndShadowing(t *testing.T) {
	result := resolutionTestFile(t, "resolution_semantics.clj")

	alias := resolutionSymbolAt(t, result.RichRoots, 5, "str/trim")
	if alias.Resolution == nil || alias.Resolution.CanonicalName != "clojure.string/trim" {
		t.Fatalf("alias resolution = %#v, want clojure.string/trim", alias.Resolution)
	}
	if alias.Resolution.Kind != reader.ResolutionNamespaceVar {
		t.Fatalf("alias resolution kind = %q, want namespace-var", alias.Resolution.Kind)
	}

	refer := resolutionSymbolAt(t, result.RichRoots, 5, "blank?")
	if refer.Resolution == nil || refer.Resolution.CanonicalName != "clojure.string/blank?" {
		t.Fatalf("refer resolution = %#v, want clojure.string/blank?", refer.Resolution)
	}

	shadowed := resolutionSymbolAt(t, result.RichRoots, 8, "trim")
	if shadowed.Resolution == nil || shadowed.Resolution.Kind != reader.ResolutionLocal || !shadowed.Resolution.Lexical {
		t.Fatalf("shadowed resolution = %#v, want lexical local", shadowed.Resolution)
	}
}

func TestResolutionSemanticsSequentialBindings(t *testing.T) {
	result := resolutionTestFile(t, "resolution_semantics.clj")

	initializer := resolutionSymbolAt(t, result.RichRoots, 11, "earlier")
	parameter := resolutionSymbolAt(t, result.RichRoots, 10, "earlier")
	if initializer.ResolvedDefinition != parameter {
		t.Fatalf("future let binding leaked into initializer: definition=%#v parameter=%#v", initializer.ResolvedDefinition, parameter)
	}
	if initializer.Resolution == nil || initializer.Resolution.Kind != reader.ResolutionLocal || !initializer.Resolution.Lexical {
		t.Fatalf("initializer resolution = %#v, want outer lexical parameter", initializer.Resolution)
	}

	body := resolutionSymbolAt(t, result.RichRoots, 13, "earlier")
	if body.ResolvedDefinition == nil || body.ResolvedDefinition == parameter {
		t.Fatalf("body did not resolve to inner let binding: definition=%#v parameter=%#v", body.ResolvedDefinition, parameter)
	}
}

func TestResolutionSemanticsExposesProvenLocalLiteralFacts(t *testing.T) {
	result := resolutionTestFile(t, "resolution_semantics.clj")
	body := resolutionSymbolAt(t, result.RichRoots, 13, "earlier")
	facts := semantics.ForNode(body, map[string]interface{}{
		"semantic-options":     map[string]bool{"type-inference": true},
		"semantic-facts-cache": semantics.NewFactsCache(),
	})
	if !facts.Constant || facts.Type != semantics.TypeKeyword {
		t.Fatalf("expected resolved let binding to expose literal keyword facts, got %#v", facts)
	}
	if facts.SemanticEvidence != semantics.EvidenceProven || facts.EvidenceSource != semantics.SourceLocalBinding {
		t.Fatalf("expected local binding provenance, got %#v", facts)
	}
}

func TestResolutionSemanticsConditionalAndLoopBindings(t *testing.T) {
	result := resolutionTestFile(t, "resolution_binding_forms.clj")

	thenBinding := resolutionSymbolAt(t, result.RichRoots, 5, "bound")
	if thenBinding.Resolution == nil || thenBinding.Resolution.Kind != reader.ResolutionLocal || thenBinding.ResolvedDefinition == nil {
		t.Fatalf("if-let then binding resolution = %#v, definition=%#v", thenBinding.Resolution, thenBinding.ResolvedDefinition)
	}

	elseValue := resolutionSymbolAt(t, result.RichRoots, 6, "value")
	if elseValue.Resolution == nil || elseValue.Resolution.Kind != reader.ResolutionLocal {
		t.Fatalf("if-let else value resolution = %#v, want outer local", elseValue.Resolution)
	}
	if elseValue.ResolvedDefinition == thenBinding.ResolvedDefinition {
		t.Fatal("if-let binding leaked into the else branch")
	}

	whenBinding := resolutionSymbolAt(t, result.RichRoots, 8, "also-bound")
	if whenBinding.Resolution == nil || whenBinding.Resolution.Kind != reader.ResolutionLocal || whenBinding.ResolvedDefinition == nil {
		t.Fatalf("when-let binding resolution = %#v, definition=%#v", whenBinding.Resolution, whenBinding.ResolvedDefinition)
	}

	loopBinding := resolutionSymbolAt(t, result.RichRoots, 12, "current")
	if loopBinding.Resolution == nil || loopBinding.Resolution.Kind != reader.ResolutionLocal || loopBinding.ResolvedDefinition == nil {
		t.Fatalf("loop binding resolution = %#v, definition=%#v", loopBinding.Resolution, loopBinding.ResolvedDefinition)
	}
}

func TestResolutionSemanticsDestructuredLiteralFacts(t *testing.T) {
	result := resolutionTestFile(t, "resolution_binding_forms.clj")
	context := map[string]interface{}{
		"semantic-options":     map[string]bool{"type-inference": true},
		"semantic-facts-cache": semantics.NewFactsCache(),
	}
	for _, tc := range []struct {
		value string
		want  semantics.AbstractType
	}{
		{value: "items", want: semantics.TypeVector},
		{value: "first-item", want: semantics.TypeNumber},
		{value: "second-item", want: semantics.TypeNumber},
		{value: "record", want: semantics.TypeMap},
	} {
		node := resolutionSymbolAt(t, result.RichRoots, 19, tc.value)
		facts := semantics.ForNode(node, context)
		if !facts.Constant || facts.Type != tc.want || facts.EvidenceSource != semantics.SourceLocalBinding {
			t.Errorf("%s facts = %#v, want proven constant %q local binding", tc.value, facts, tc.want)
		}
	}

	unknown := resolutionSymbolAt(t, result.RichRoots, 23, "items")
	unknownFacts := semantics.ForNode(unknown, context)
	if unknownFacts.Constant || unknownFacts.EvidenceSource == semantics.SourceLocalBinding {
		t.Fatalf("unknown destructured source must remain unpromoted, got %#v", unknownFacts)
	}
}

func TestResolutionSemanticsDestructuring(t *testing.T) {
	result := resolutionTestFile(t, "resolution_semantics.clj")
	for _, value := range []string{"name", "user", "first-value", "second-value", "rest-values"} {
		node := resolutionSymbolAt(t, result.RichRoots, 16, value)
		if node.Resolution == nil || node.Resolution.Kind != reader.ResolutionLocal || !node.Resolution.Lexical {
			t.Fatalf("destructured %q resolution = %#v, want lexical local", value, node.Resolution)
		}
	}
}

func TestResolutionSemanticsClojureScriptAndCLJC(t *testing.T) {
	for _, tc := range []struct {
		name string
		line int
	}{
		{name: "resolution_semantics.cljs", line: 5},
		{name: "resolution_semantics.cljc", line: 6},
	} {
		result := resolutionTestFile(t, tc.name)
		alias := resolutionSymbolAt(t, result.RichRoots, tc.line, "str/trim")
		if alias.Resolution == nil || alias.Resolution.CanonicalName != "clojure.string/trim" || alias.Resolution.Kind != reader.ResolutionNamespaceVar {
			t.Fatalf("%s alias resolution = %#v, want clojure.string/trim namespace-var", tc.name, alias.Resolution)
		}
	}
}

func TestResolutionSemanticsAmbiguousCLJCRemainsUnresolved(t *testing.T) {
	result := resolutionTestFile(t, "resolution_semantics_ambiguous.cljc")
	alias := resolutionSymbolAt(t, result.RichRoots, 6, "str/trim")
	if alias.Resolution == nil || alias.Resolution.Kind != reader.ResolutionUnresolved || alias.Resolution.NamespaceKnown {
		t.Fatalf("ambiguous .cljc alias resolution = %#v, want unresolved with unknown namespace", alias.Resolution)
	}
}

func TestResolutionSemanticsUnknownQuotedSymbolRemainsUnresolved(t *testing.T) {
	result := resolutionTestFile(t, "resolution_semantics.clj")
	quoted := resolutionSymbolAt(t, result.RichRoots, 19, "missing-call")
	if quoted.Resolution == nil || quoted.Resolution.Kind != reader.ResolutionUnresolved {
		t.Fatalf("quoted symbol resolution = %#v, want unresolved", quoted.Resolution)
	}
}
