package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestCaseWithNonLiteralTestValues(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "case_with_non_literal_test_values.clj",
		RuleID:        "case-with-non-literal-test-values",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "non-literal symbol", StartLine: 15},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 5}, {StartLine: 6}, {StartLine: 7}, {StartLine: 8}, {StartLine: 9},
			{StartLine: 20}, {StartLine: 27},
		},
	})
}
