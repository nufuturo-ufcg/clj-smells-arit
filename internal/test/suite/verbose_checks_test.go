package suite

import (
	"testing"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestVerboseChecks(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "verbose_checks.clj",
			RuleID:        "verbose-checks",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "= 0 x", StartLine: 4},
				{Message: "+ 1 x", StartLine: 10},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
