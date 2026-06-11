package framework

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	rules "github.com/thlaurentino/arit/internal/rules"
)

type ExpectedFinding struct {
	Message   string
	StartLine int
}

type RuleTestCase struct {
	FileToAnalyze    string
	RuleID           string
	ExpectedFindings []ExpectedFinding
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

	assert.Len(t, filteredFindings, len(tc.ExpectedFindings),
		"Incorrect number of findings for rule '%s'", tc.RuleID)

	actualFindings := make(map[int][]rules.Finding)
	for _, f := range filteredFindings {
		actualFindings[f.Location.StartLine] = append(actualFindings[f.Location.StartLine], f)
	}

	matchedIndices := make(map[string]bool)

	for i, expected := range tc.ExpectedFindings {
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
						assert.Equal(t, tc.RuleID, f.RuleID,
							"Incorrect RuleID for finding on line %d", expected.StartLine)
						break
					}
				}
				assert.True(t, matched,
					"Expected finding on line %d with message containing %q, but none matched", expected.StartLine, expected.Message)
			}
		})
	}
}
