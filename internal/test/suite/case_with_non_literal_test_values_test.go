package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestCaseWithNonLiteralTestValues(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "case_with_non_literal_test_values.clj",
		RuleID:        "case-with-non-literal-test-values",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "local symbol", StartLine: 28, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "local var `global-constant`", StartLine: 16, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 5}, {StartLine: 6}, {StartLine: 7}, {StartLine: 8}, {StartLine: 9},
			{StartLine: 13}, {StartLine: 22}, {StartLine: 24}, {StartLine: 27}, {StartLine: 35}, {StartLine: 42},
		},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "case_with_non_literal_global_var.clj",
		RuleID:        "case-with-non-literal-test-values",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "local var `status`", StartLine: 8, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "local var `mode`", StartLine: 13, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 19},
			{StartLine: 24},
		},
	})
}
