package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestConditionalBuildup(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "conditional_build_up.clj",
			RuleID:       "conditional-build-up",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "cond->", StartLine: 7},  
				{Message: "cond->", StartLine: 13}, 
				{Message: "cond->", StartLine: 20},
				{Message: "cond->", StartLine: 28},
				{Message: "cond->", StartLine: 36},
				{Message: "cond->", StartLine: 45},
				{Message: "cond->", StartLine: 53},
				{Message: "cond->", StartLine: 60},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
