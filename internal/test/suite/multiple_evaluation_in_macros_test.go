package suite

import (
	"testing"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMultipleEvaluationInMacros(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "multiple_evaluation_in_macros.clj",
			RuleID:        "multiple-evaluation-in-macros",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "The macro bad-macro presents multiple calls to the input arguments x", StartLine: 3},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
