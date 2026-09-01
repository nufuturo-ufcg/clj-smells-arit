package clojurespecific

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/rules"
)

func TestCatalogThreadingFalseNegativesNowProduceFindings(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the regression test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	files := []string{
		"complex_01.clj",
		"complex_04.clj",
		"example_05.clj",
		"example_09.clj",
		"example_13.clj",
		"example_15.clj",
	}
	cfg := &config.Config{
		EnabledRules:  map[string]bool{"misused-threading": true},
		EnabledGroups: map[string]bool{},
		RuleConfig:    map[string]config.RuleSettings{},
	}
	instance := analyzer.NewAnalyzer(cfg)
	for _, name := range files {
		path := filepath.Join(root, "docs", "expanded_smells_catalog", "30_misused_threading", name)
		result, err := instance.AnalyzeFile(path)
		if err != nil {
			t.Fatalf("analyze %s: %v", name, err)
		}
		found := false
		for _, finding := range result.Findings {
			if finding.RuleID == "misused-threading" {
				found = true
				if finding.Confidence != rules.ConfidenceProven {
					t.Errorf("%s: confidence = %q, want proven", name, finding.Confidence)
				}
			}
		}
		if !found {
			t.Errorf("%s: expected a misused-threading finding", name)
		}
	}
}

func TestMapWithNilValuesReportsContractEvidence(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the regression test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	path := filepath.Join(root, "internal", "test", "data", "map_with_nil_values.clj")
	cfg := &config.Config{
		EnabledRules:  map[string]bool{"map-with-nil-values": true},
		EnabledGroups: map[string]bool{},
		RuleConfig:    map[string]config.RuleSettings{"map-with-nil-values": {"strict": false}},
	}
	result, err := analyzer.NewAnalyzer(cfg).AnalyzeFile(path)
	if err != nil {
		t.Fatalf("analyze map fixture: %v", err)
	}
	findings := make([]rules.Finding, 0)
	for _, finding := range result.Findings {
		if finding.RuleID == "map-with-nil-values" {
			findings = append(findings, finding)
		}
	}
	if len(findings) != 2 {
		t.Fatalf("expected two map findings, got %d", len(findings))
	}
	for _, finding := range findings {
		if finding.Confidence != rules.ConfidenceContextual {
			t.Errorf("expected contextual confidence, got %q", finding.Confidence)
		}
		if len(finding.MissingEvidence) == 0 || finding.MissingEvidence[0] == "external-contract" {
			t.Errorf("expected specific missing evidence, got %#v", finding.MissingEvidence)
		}
	}
}
