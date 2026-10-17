package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestConditionalBuildup(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "conditional_build_up.clj",
			RuleID:        "conditional-build-up",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "cond->", StartLine: 13, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "cond->", StartLine: 20, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "cond->", StartLine: 36, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 119}, {StartLine: 124},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
