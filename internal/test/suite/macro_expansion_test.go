package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMacroExpansion(t *testing.T) {
	// Enable the experimental flag specifically for this test
	originalFlag := analyzer.EnableExperimentalMacroExpansion
	analyzer.EnableExperimentalMacroExpansion = true
	defer func() { analyzer.EnableExperimentalMacroExpansion = originalFlag }()

	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "macro_expansion.clj",
		RuleID:        "unnecessary-into",
		ExpectedFindings: []framework.ExpectedFinding{
			{StartLine: 4},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 9},
		},
	})
}
