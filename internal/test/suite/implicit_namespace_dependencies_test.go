package suite

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestImplicitNamespaceDependencies(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "implicit_namespace_dependencies.clj",
			RuleID:        "implicit-namespace-dependencies",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Implicit namespace dependency: :use directive", StartLine: 2, Severity: rules.SeverityHint, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Implicit namespace dependency: :use directive", StartLine: 3, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Implicit namespace dependency: :refer :all", StartLine: 5, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Implicit namespace dependency: standalone (use", StartLine: 10, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 27},
			},
		},
		{
			FileToAnalyze: "implicit_namespace_dependencies.cljs",
			RuleID:        "implicit-namespace-dependencies",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Implicit namespace dependency: :refer :all", StartLine: 2, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 7},
				{StartLine: 8},
				{StartLine: 12},
				{StartLine: 16},
				{StartLine: 20},
			},
		},
		{
			FileToAnalyze:     "implicit_namespace_dependencies_interop.cljs",
			RuleID:            "implicit-namespace-dependencies",
			ExpectedFindings:  []framework.ExpectedFinding{},
			ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 5}, {StartLine: 8}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}

func TestImplicitNamespaceDependenciesWithProjectIndex(t *testing.T) {
	testFile, err := filepath.Abs(filepath.Join("../data", "implicit_namespace_dependencies.cljs"))
	assert.NoError(t, err)

	enabledRules := make(map[string]bool)
	for _, rule := range rules.AllRules() {
		enabledRules[rule.Meta().ID] = false
	}
	enabledRules["implicit-namespace-dependencies"] = true

	testConfig := &config.Config{
		EnabledRules: enabledRules,
		RuleConfig:   make(map[string]config.RuleSettings),
	}
	result, err := analyzer.NewAnalyzer(testConfig).AnalyzeFileWithProjectIndex(testFile, semantics.NewProjectIndex())
	assert.NoError(t, err)

	var findings []rules.Finding
	for _, finding := range result.Findings {
		if finding.RuleID == "implicit-namespace-dependencies" {
			findings = append(findings, finding)
		}
	}
	assert.Len(t, findings, 2)

	byLine := make(map[int]rules.Finding)
	for _, finding := range findings {
		byLine[finding.Location.StartLine] = finding
	}

	referAll, ok := byLine[2]
	assert.True(t, ok)
	assert.Contains(t, referAll.Message, "Implicit namespace dependency: :refer :all")
	assert.Equal(t, rules.ConfidenceContextual, referAll.Confidence)

	qualified, ok := byLine[20]
	assert.True(t, ok)
	assert.Contains(t, qualified.Message, "Qualified namespace dependency")
	assert.Equal(t, rules.ConfidenceContextual, qualified.Confidence)
	assert.Equal(t, []string{"namespace-loading-contract", "project-dependency-index"}, qualified.MissingEvidence)
}

func TestImplicitNamespaceDependenciesWithMultiFileProjectIndex(t *testing.T) {
	dataDir, err := filepath.Abs("../data")
	assert.NoError(t, err)

	providerPath := filepath.Join(dataDir, "implicit_namespace_dependencies_project_provider.clj")
	explicitPath := filepath.Join(dataDir, "implicit_namespace_dependencies_project_explicit.clj")
	missingPath := filepath.Join(dataDir, "implicit_namespace_dependencies_project_missing.clj")

	projectIndex := semantics.NewProjectIndex()
	for _, path := range []string{providerPath, explicitPath, missingPath} {
		assert.NoError(t, projectIndex.IndexFile(path))
	}
	if _, ok := projectIndex.DefinitionOf("project.runtime/start!"); !ok {
		t.Fatal("expected project index to contain project.runtime/start!")
	}

	enabledRules := make(map[string]bool)
	for _, rule := range rules.AllRules() {
		enabledRules[rule.Meta().ID] = false
	}
	enabledRules["implicit-namespace-dependencies"] = true
	testConfig := &config.Config{
		EnabledRules: enabledRules,
		RuleConfig:   make(map[string]config.RuleSettings),
	}
	analyzerInstance := analyzer.NewAnalyzer(testConfig)

	explicitResult, err := analyzerInstance.AnalyzeFileWithProjectIndex(explicitPath, projectIndex)
	assert.NoError(t, err)
	assert.Empty(t, findingsForRule(explicitResult, "implicit-namespace-dependencies"))

	missingResult, err := analyzerInstance.AnalyzeFileWithProjectIndex(missingPath, projectIndex)
	assert.NoError(t, err)
	findings := findingsForRule(missingResult, "implicit-namespace-dependencies")
	assert.Len(t, findings, 1)
	if len(findings) == 1 {
		assert.Contains(t, findings[0].Message, "Qualified namespace dependency")
		assert.Equal(t, rules.ConfidenceContextual, findings[0].Confidence)
		assert.Equal(t, []string{"namespace-loading-contract", "project-dependency-index"}, findings[0].MissingEvidence)
	}
}

func findingsForRule(result analyzer.AnalysisResult, ruleID string) []rules.Finding {
	findings := make([]rules.Finding, 0)
	for _, finding := range result.Findings {
		if finding.RuleID == ruleID {
			findings = append(findings, finding)
		}
	}
	return findings
}
