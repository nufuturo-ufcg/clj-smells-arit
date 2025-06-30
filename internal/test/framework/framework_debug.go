package framework

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/rules"
)

func DebugRuleTest(t *testing.T, tc RuleTestCase) {
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

	fmt.Printf("\n--- DEBUG: Findings for rule '%s' ---\n", tc.RuleID)
	fmt.Printf("Total findings: %d\n", len(filteredFindings))

	if len(filteredFindings) == 0 {
		fmt.Printf("\nNo findings found!\n")
		fmt.Printf("   Please check if the rule '%s' is implemented correctly\n", tc.RuleID)
		fmt.Printf("   and if the test file contains code that should be detected.\n")
	}

	for i, finding := range filteredFindings {
		fmt.Printf("\nFinding %d:\n", i+1)
		fmt.Printf("   Line: %d\n", finding.Location.StartLine)
		fmt.Printf("   Message: %q\n", finding.Message)
		fmt.Printf("   Suggested for test:\n")
		fmt.Printf("      {Message: %q, StartLine: %d},\n",
			extractKeyPhrase(finding.Message), finding.Location.StartLine)
	}
	fmt.Printf("\n--- END DEBUG ---\n\n")
}

func extractKeyPhrase(message string) string {

	if len(message) > 50 {

		if idx := findPattern(message, "(", ")"); idx != "" {
			return idx
		}
		if idx := findPattern(message, "'", "'"); idx != "" {
			return idx
		}

		return message[:30] + "..."
	}
	return message
}

func findPattern(text, start, end string) string {
	startIdx := -1
	endIdx := -1

	for i := 0; i < len(text)-len(start); i++ {
		if text[i:i+len(start)] == start {
			startIdx = i + len(start)
			break
		}
	}

	if startIdx == -1 {
		return ""
	}

	for i := startIdx; i < len(text)-len(end); i++ {
		if text[i:i+len(end)] == end {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return ""
	}

	return text[startIdx:endIdx]
}
