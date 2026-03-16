package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestConditionalBuildupLet(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "conditional_build_up.clj",
			RuleID:       "conditional-build-up",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "cond->", StartLine: 9},  
				{Message: "cond->", StartLine: 26}, 
				{Message: "cond->", StartLine: 33},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
