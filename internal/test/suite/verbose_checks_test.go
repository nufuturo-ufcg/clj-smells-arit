package suite

import (
	"github.com/thlaurentino/arit/internal/test/framework"
	"testing"
)

func TestVerboseChecks(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "verbose_checks.clj",
			RuleID:        "verbose-checks",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "= 0 x", StartLine: 4},
				{Message: "= x true", StartLine: 6},
				{Message: "= nil x", StartLine: 8},
				{Message: "+ 1 x", StartLine: 10},
				{Message: ">= x 0", StartLine: 14},
				{Message: "= 0 (long ...)", StartLine: 25},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 17},
				{StartLine: 21},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
