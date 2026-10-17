package framework

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	rules "github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"

	_ "github.com/thlaurentino/arit/internal/rules/clojurespecific"
	_ "github.com/thlaurentino/arit/internal/rules/functional"
	_ "github.com/thlaurentino/arit/internal/rules/traditional"
)

type ExpectedFinding struct {
	Message                 string
	StartLine               int
	Severity                rules.Severity
	RequireConfidence       rules.Confidence
	RequireContextual       bool
	RequireNonContextual    bool
	ExpectedMissingEvidence []string
}

type RuleTestCase struct {
	FileToAnalyze     string
	RuleID            string
	ExpectedFindings  []ExpectedFinding
	ForbiddenFindings []ExpectedFinding
	AssertNoFindings  bool
}

type ProjectTestCase struct {
	Files             map[string]string // relative path -> file content
	DataFiles         []string          // relative filenames in ../data to index
	EntryPoint        string            // relative filename of the entrypoint file to analyze
	RuleID            string
	ExpectedFindings  []ExpectedFinding
	ForbiddenFindings []ExpectedFinding
	AssertNoFindings  bool
}

func RunRuleTest(t *testing.T, tc RuleTestCase) {
	t.Helper()

	allRules := rules.AllRules()
	enabledRules := make(map[string]bool)
	for _, rule := range allRules {
		enabledRules[rule.Meta().ID] = false
	}

	enabledRules[tc.RuleID] = true

	testConfig := &config.Config{
		EnabledRules: enabledRules,
		RuleConfig:   make(map[string]config.RuleSettings),
	}

	testFile, err := filepath.Abs(filepath.Join("../data", tc.FileToAnalyze))
	assert.NoError(t, err, "Failed to get absolute path for test file")

	result, err := analyzer.AnalyzeFile(testFile, testConfig)
	assert.NoError(t, err, "Failed to analyze test file")

	var filteredFindings []rules.Finding
	for _, finding := range result.Findings {
		if finding.RuleID == tc.RuleID {
			filteredFindings = append(filteredFindings, finding)
		}
	}

	if tc.AssertNoFindings {
		assert.Empty(t, filteredFindings, "Expected 0 findings for rule '%s', but got %d", tc.RuleID, len(filteredFindings))
		return
	}

	assert.Len(t, filteredFindings, len(tc.ExpectedFindings),
		"Incorrect number of findings for rule '%s'", tc.RuleID)

	assertFindings(t, tc.RuleID, filteredFindings, tc.ExpectedFindings, tc.ForbiddenFindings)
}

func RunProjectTest(t *testing.T, tc ProjectTestCase) {
	t.Helper()

	tempDir := t.TempDir()
	projectIndex := semantics.NewProjectIndex()

	var allFilePaths []string
	for relPath, content := range tc.Files {
		fullPath := filepath.Join(tempDir, relPath)
		assert.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		assert.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
		allFilePaths = append(allFilePaths, fullPath)
	}

	for _, dataFile := range tc.DataFiles {
		fullPath, err := filepath.Abs(filepath.Join("../data", dataFile))
		assert.NoError(t, err)
		allFilePaths = append(allFilePaths, fullPath)
	}

	for _, path := range allFilePaths {
		assert.NoError(t, projectIndex.IndexFile(path), "Failed to index file %s", path)
	}

	entryPointPath := filepath.Join(tempDir, tc.EntryPoint)
	if _, err := os.Stat(entryPointPath); os.IsNotExist(err) {
		if candidate, err := filepath.Abs(filepath.Join("../data", tc.EntryPoint)); err == nil {
			if _, statErr := os.Stat(candidate); statErr == nil {
				entryPointPath = candidate
			}
		}
	}

	allRules := rules.AllRules()
	enabledRules := make(map[string]bool)
	for _, rule := range allRules {
		enabledRules[rule.Meta().ID] = false
	}
	enabledRules[tc.RuleID] = true

	testConfig := &config.Config{
		EnabledRules: enabledRules,
		RuleConfig:   make(map[string]config.RuleSettings),
	}

	analyzerInstance := analyzer.NewAnalyzer(testConfig)
	result, err := analyzerInstance.AnalyzeFileWithProjectIndex(entryPointPath, projectIndex)
	assert.NoError(t, err, "Failed to analyze entrypoint %s", entryPointPath)

	var filteredFindings []rules.Finding
	for _, finding := range result.Findings {
		if finding.RuleID == tc.RuleID {
			filteredFindings = append(filteredFindings, finding)
		}
	}

	if tc.AssertNoFindings {
		assert.Empty(t, filteredFindings, "Expected 0 findings for rule '%s', but got %d", tc.RuleID, len(filteredFindings))
		return
	}

	assert.Len(t, filteredFindings, len(tc.ExpectedFindings),
		"Incorrect number of findings for rule '%s'", tc.RuleID)

	assertFindings(t, tc.RuleID, filteredFindings, tc.ExpectedFindings, tc.ForbiddenFindings)
}

func assertFindings(t *testing.T, ruleID string, filteredFindings []rules.Finding, expectedFindings []ExpectedFinding, forbiddenFindings []ExpectedFinding) {
	t.Helper()

	actualFindings := make(map[int][]rules.Finding)
	for _, f := range filteredFindings {
		actualFindings[f.Location.StartLine] = append(actualFindings[f.Location.StartLine], f)
	}

	matchedIndices := make(map[string]bool)

	for i, expected := range expectedFindings {
		t.Run(fmt.Sprintf("Finding_%d_line_%d", i+1, expected.StartLine), func(t *testing.T) {
			findingsOnLine, found := actualFindings[expected.StartLine]
			assert.True(t, found,
				"Expected finding on line %d, but none was found", expected.StartLine)

			if found {
				var matched bool
				for idx, f := range findingsOnLine {
					key := fmt.Sprintf("%d_%d", expected.StartLine, idx)
					if !matchedIndices[key] && strings.Contains(f.Message, expected.Message) {
						matchedIndices[key] = true
						matched = true
						assert.Equal(t, ruleID, f.RuleID,
							"Incorrect RuleID for finding on line %d", expected.StartLine)
						if expected.Severity != "" {
							assert.Equal(t, expected.Severity, f.Severity,
								"Incorrect severity for finding on line %d", expected.StartLine)
						}
						if expected.RequireConfidence != "" {
							assert.Equal(t, expected.RequireConfidence, f.Confidence,
								"Incorrect confidence for finding on line %d", expected.StartLine)
						}
						if expected.RequireContextual {
							assert.True(t, f.Contextual, "Expected contextual finding on line %d", expected.StartLine)
						}
						if expected.RequireNonContextual {
							assert.False(t, f.Contextual, "Expected non-contextual finding on line %d", expected.StartLine)
						}
						if len(expected.ExpectedMissingEvidence) > 0 {
							assert.True(t, f.Contextual, "Expected finding on line %d to be contextual when ExpectedMissingEvidence is specified", expected.StartLine)
							for _, requiredEvidence := range expected.ExpectedMissingEvidence {
								assert.Contains(t, f.MissingEvidence, requiredEvidence,
									"Finding on line %d missing expected evidence %q; actual missing evidence: %#v", expected.StartLine, requiredEvidence, f.MissingEvidence)
							}
						}
						break
					}
				}
				assert.True(t, matched,
					"Expected finding on line %d with message containing %q, but none matched", expected.StartLine, expected.Message)
			}
		})
	}

	for i, forbidden := range forbiddenFindings {
		t.Run(fmt.Sprintf("Forbidden_%d_line_%d", i+1, forbidden.StartLine), func(t *testing.T) {
			for _, finding := range actualFindings[forbidden.StartLine] {
				if forbidden.Message == "" || strings.Contains(finding.Message, forbidden.Message) {
					t.Errorf("Unexpected finding on line %d: %s", forbidden.StartLine, finding.Message)
				}
			}
		})
	}
}
