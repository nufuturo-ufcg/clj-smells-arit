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

func TestMultipleEvaluationInMacros(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "multiple_evaluation_in_macros.clj",
			RuleID:        "multiple-evaluation-in-macros",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "The macro double-eval-basic presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 6, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro square-unhygienic presents multiple calls to the input arguments x without defining temporary local variables.", StartLine: 12, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro log-and-run presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 16, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro bad-bindings presents multiple calls to the input arguments val-expr without defining temporary local variables.", StartLine: 22, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro repeat-eval-loop presents multiple calls to the input arguments coll-expr without defining temporary local variables.", StartLine: 28, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro try-re-eval presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 34, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro pair-value presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 41, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro unpack-twice presents multiple calls to the input arguments items-expr without defining temporary local variables.", StartLine: 46, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro false-safe-gensym presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 50, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro assert-verbose presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 56, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro duplicate-body presents multiple calls to the input arguments body without defining temporary local variables.", StartLine: 140, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "The macro one-risky-arity presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 144, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 86},
				{StartLine: 108},
				{StartLine: 114},
				{StartLine: 121},
				{StartLine: 127},
				{StartLine: 132},
				{StartLine: 136},
				{StartLine: 149},
				{StartLine: 155},
				{StartLine: 161},
				{StartLine: 167},
				{StartLine: 174},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}

	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "multiple_evaluation_in_macros.cljc",
		RuleID:        "multiple-evaluation-in-macros",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "The macro duplicated-cljc presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 3, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 6},
		},
	})
}

func TestMultipleEvaluationInMacrosContextualFindingsHaveEvidence(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the regression test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	path := filepath.Join(root, "internal", "test", "data", "multiple_evaluation_in_macros.clj")
	cfg := &config.Config{
		EnabledRules:  map[string]bool{"multiple-evaluation-in-macros": true},
		EnabledGroups: map[string]bool{},
		RuleConfig:    map[string]config.RuleSettings{},
	}
	result, err := analyzer.NewAnalyzer(cfg).AnalyzeFile(path)
	if err != nil {
		t.Fatalf("analyze fixture: %v", err)
	}

	count := 0
	for _, finding := range result.Findings {
		if finding.RuleID != "multiple-evaluation-in-macros" {
			continue
		}
		count++
		if finding.Confidence != rules.ConfidenceContextual || !finding.Contextual {
			t.Errorf("line %d: expected contextual finding, got confidence=%q contextual=%t", finding.Location.StartLine, finding.Confidence, finding.Contextual)
		}
		if finding.ContextualReason == "" {
			t.Errorf("line %d: contextual finding has no reason", finding.Location.StartLine)
		}
		if len(finding.MissingEvidence) == 0 {
			t.Errorf("line %d: contextual finding has no missing evidence", finding.Location.StartLine)
		}
	}
	if count != 12 {
		t.Fatalf("expected 12 contextual findings, got %d", count)
	}
}

func TestMultipleEvaluationInMacrosOptInPromotionRequiresEffectfulCallSite(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the regression test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	providerPath := filepath.Join(root, "internal", "test", "data", "macro_call_site_provider.clj")
	consumerPath := filepath.Join(root, "internal", "test", "data", "macro_call_site_consumer.clj")

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
		EnabledRules: map[string]bool{"multiple-evaluation-in-macros": true},
		RuleConfig: map[string]config.RuleSettings{
			"multiple-evaluation-in-macros": {
				"proven-macros": []interface{}{"macro-call-site.provider/duplicated"},
			},
		},
	}
	result, err := analyzer.NewAnalyzer(cfg).AnalyzeFileWithProjectIndex(providerPath, index)
	if err != nil {
		t.Fatalf("analyze provider: %v", err)
	}
	var finding *rules.Finding
	for index := range result.Findings {
		if result.Findings[index].RuleID == "multiple-evaluation-in-macros" {
			finding = &result.Findings[index]
			break
		}
	}
	if finding == nil {
		t.Fatalf("expected one promoted finding, got %#v", result.Findings)
	}
	if finding.Confidence != rules.ConfidenceProven || finding.Contextual {
		t.Fatalf("expected proven non-contextual finding, got confidence=%q contextual=%t", finding.Confidence, finding.Contextual)
	}
}

func TestMultipleEvaluationInMacrosDoesNotPromoteInvalidOrUnknownCallSites(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the regression test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	providerPath := filepath.Join(root, "internal", "test", "data", "macro_call_site_provider.clj")
	consumerPath := filepath.Join(root, "internal", "test", "data", "macro_call_site_invalid_consumer.clj")

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
		EnabledRules: map[string]bool{"multiple-evaluation-in-macros": true},
		RuleConfig: map[string]config.RuleSettings{
			"multiple-evaluation-in-macros": {
				"proven-macros": []interface{}{"macro-call-site.provider/duplicated"},
			},
		},
	}
	result, err := analyzer.NewAnalyzer(cfg).AnalyzeFileWithProjectIndex(providerPath, index)
	if err != nil {
		t.Fatalf("analyze provider: %v", err)
	}
	for _, finding := range result.Findings {
		if finding.RuleID != "multiple-evaluation-in-macros" {
			continue
		}
		if finding.Confidence != rules.ConfidenceContextual || !finding.Contextual {
			t.Fatalf("expected invalid/unknown calls to remain contextual, got confidence=%q contextual=%t", finding.Confidence, finding.Contextual)
		}
		return
	}
	t.Fatal("expected contextual multiple-evaluation finding")
}
