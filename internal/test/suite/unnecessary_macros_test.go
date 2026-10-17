package suite

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnnecessaryMacros(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "unnecessary_macros.clj",
		RuleID:        "unnecessary-macros",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "prefer a normal function", StartLine: 3, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "prefer a normal function", StartLine: 6, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "prefer a normal function", StartLine: 19, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "prefer a normal function", StartLine: 22, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "prefer a normal function", StartLine: 68, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "prefer a normal function", StartLine: 71, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 9}, {StartLine: 12}, {StartLine: 16}, {StartLine: 25}, {StartLine: 28}, {StartLine: 33}, {StartLine: 37},
			{StartLine: 41}, {StartLine: 44}, {StartLine: 47}, {StartLine: 50}, {StartLine: 53}, {StartLine: 56}, {StartLine: 59}, {StartLine: 62}},
	})
}

func TestUnnecessaryMacrosOptInPromotionRequiresProvenIndexedArguments(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the regression test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	providerPath := filepath.Join(root, "internal", "test", "data", "unnecessary_macro_call_site_provider.clj")
	consumerPath := filepath.Join(root, "internal", "test", "data", "unnecessary_macro_call_site_consumer.clj")
	index := semantics.NewProjectIndex()
	if err := index.IndexFile(providerPath); err != nil {
		t.Fatalf("index provider: %v", err)
	}
	if err := index.IndexFile(consumerPath); err != nil {
		t.Fatalf("index consumer: %v", err)
	}
	index.BuildMacroCallSites()
	index.MarkMacroCallSitesComplete()
	cfg := &config.Config{
		EnabledRules: map[string]bool{"unnecessary-macros": true},
		RuleConfig: map[string]config.RuleSettings{
			"unnecessary-macros": {
				"proven-macros": []interface{}{"unnecessary-macro-call-site.provider/add-one"},
			},
		},
	}
	result, err := analyzer.NewAnalyzer(cfg).AnalyzeFileWithProjectIndex(providerPath, index)
	if err != nil {
		t.Fatalf("analyze provider: %v", err)
	}
	var finding *rules.Finding
	for i := range result.Findings {
		if result.Findings[i].RuleID == "unnecessary-macros" {
			finding = &result.Findings[i]
			break
		}
	}
	if finding == nil {
		t.Fatalf("expected unnecessary-macros finding, got %#v", result.Findings)
	}
	if finding.Confidence != rules.ConfidenceProven || finding.Contextual {
		t.Fatalf("expected proven non-contextual finding, got confidence=%q contextual=%t", finding.Confidence, finding.Contextual)
	}
}

func TestUnnecessaryMacrosDoesNotPromoteUnknownIndexedArguments(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the regression test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	providerPath := filepath.Join(root, "internal", "test", "data", "unnecessary_macro_call_site_provider.clj")
	consumerPath := filepath.Join(root, "internal", "test", "data", "unnecessary_macro_call_site_invalid_consumer.clj")
	index := semantics.NewProjectIndex()
	if err := index.IndexFile(providerPath); err != nil {
		t.Fatalf("index provider: %v", err)
	}
	if err := index.IndexFile(consumerPath); err != nil {
		t.Fatalf("index invalid consumer: %v", err)
	}
	index.BuildMacroCallSites()
	index.MarkMacroCallSitesComplete()
	cfg := &config.Config{
		EnabledRules: map[string]bool{"unnecessary-macros": true},
		RuleConfig: map[string]config.RuleSettings{
			"unnecessary-macros": {
				"proven-macros": []interface{}{"unnecessary-macro-call-site.provider/add-one"},
			},
		},
	}
	result, err := analyzer.NewAnalyzer(cfg).AnalyzeFileWithProjectIndex(providerPath, index)
	if err != nil {
		t.Fatalf("analyze provider: %v", err)
	}
	for _, finding := range result.Findings {
		if finding.RuleID != "unnecessary-macros" {
			continue
		}
		if finding.Confidence != rules.ConfidenceContextual || !finding.Contextual {
			t.Fatalf("expected unknown call argument to remain contextual, got confidence=%q contextual=%t", finding.Confidence, finding.Contextual)
		}
		return
	}
	t.Fatal("expected contextual unnecessary-macros finding")
}

func TestUnnecessaryMacrosSupportsRequireMacrosAndAllExercisedArities(t *testing.T) {
	root := testRepoRoot(t)
	providerPath := filepath.Join(root, "internal", "test", "data", "unnecessary_macro_require_provider.clj")
	consumerPath := filepath.Join(root, "internal", "test", "data", "unnecessary_macro_require_consumer.clj")
	index := indexMacroFiles(t, providerPath, consumerPath)
	result := analyzeConfiguredMacro(t, providerPath, index)
	finding := findUnnecessaryMacroFinding(result.Findings)
	if finding == nil || finding.Confidence != rules.ConfidenceProven || finding.Contextual {
		t.Fatalf("expected all exercised require-macros arities to promote, got %#v", result.Findings)
	}
}

func TestUnnecessaryMacrosKeepsUnexercisedOrIndirectAritiesContextual(t *testing.T) {
	root := testRepoRoot(t)
	providerPath := filepath.Join(root, "internal", "test", "data", "unnecessary_macro_require_provider.clj")
	for _, consumerName := range []string{"unnecessary_macro_require_partial.clj", "unnecessary_macro_require_indirect.clj"} {
		t.Run(consumerName, func(t *testing.T) {
			consumerPath := filepath.Join(root, "internal", "test", "data", consumerName)
			index := indexMacroFiles(t, providerPath, consumerPath)
			result := analyzeConfiguredMacro(t, providerPath, index)
			finding := findUnnecessaryMacroFinding(result.Findings)
			if finding == nil || finding.Confidence != rules.ConfidenceContextual || !finding.Contextual {
				t.Fatalf("expected conservative contextual result, got %#v", result.Findings)
			}
		})
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the regression test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
}

func indexMacroFiles(t *testing.T, paths ...string) *semantics.ProjectIndex {
	t.Helper()
	index := semantics.NewProjectIndex()
	for _, path := range paths {
		if err := index.IndexFile(path); err != nil {
			t.Fatalf("index %s: %v", path, err)
		}
	}
	index.BuildMacroCallSites()
	index.MarkMacroCallSitesComplete()
	return index
}

func analyzeConfiguredMacro(t *testing.T, providerPath string, index *semantics.ProjectIndex) analyzer.AnalysisResult {
	t.Helper()
	cfg := &config.Config{
		EnabledRules: map[string]bool{"unnecessary-macros": true},
		RuleConfig: map[string]config.RuleSettings{
			"unnecessary-macros": {"proven-macros": []interface{}{"unnecessary-macro-require.provider/wrap"}},
		},
	}
	result, err := analyzer.NewAnalyzer(cfg).AnalyzeFileWithProjectIndex(providerPath, index)
	if err != nil {
		t.Fatalf("analyze provider: %v", err)
	}
	return result
}

func findUnnecessaryMacroFinding(findings []rules.Finding) *rules.Finding {
	for index := range findings {
		if findings[index].RuleID == "unnecessary-macros" {
			return &findings[index]
		}
	}
	return nil
}

// Names the analyzer cannot resolve are accepted as clojure.core functions
// only when the ns declaration rules out every other source for them.
func TestUnnecessaryMacrosCoreNameProvenance(t *testing.T) {
	t.Run("unresolved core function with plain ns", func(t *testing.T) {
		framework.RunProjectTest(t, framework.ProjectTestCase{
			RuleID: "unnecessary-macros",
			Files: map[string]string{
				"plain.clj": "(ns plain\n  (:require [clojure.string :as str]))\n\n(defmacro m-aget [a b]\n  `(aget ~a ~b))\n\n(defmacro m-join [sep coll]\n  `(str/join ~sep ~coll))\n",
			},
			EntryPoint: "plain.clj",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "prefer a normal function", StartLine: 4, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "prefer a normal function", StartLine: 7, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
		})
	})
	t.Run("core name replaced by a referred var", func(t *testing.T) {
		framework.RunProjectTest(t, framework.ProjectTestCase{
			RuleID: "unnecessary-macros",
			Files: map[string]string{
				"excluded.clj": "(ns excluded\n  (:refer-clojure :exclude [aget])\n  (:require [my.lib :refer [aget]]))\n\n(defmacro m-aget [a b]\n  `(aget ~a ~b))\n",
			},
			EntryPoint:       "excluded.clj",
			AssertNoFindings: true,
		})
	})
	t.Run("core name possibly supplied by :use", func(t *testing.T) {
		framework.RunProjectTest(t, framework.ProjectTestCase{
			RuleID: "unnecessary-macros",
			Files: map[string]string{
				"used.clj": "(ns used\n  (:use [my.lib]))\n\n(defmacro m-even [n]\n  `(even? ~n))\n",
			},
			EntryPoint:       "used.clj",
			AssertNoFindings: true,
		})
	})
}
