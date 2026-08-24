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
				{Message: "= x true", StartLine: 6},
				{Message: "= nil x", StartLine: 8},
				{Message: "+ 1 x", StartLine: 10},
				{Message: "= 0 (long ...)", StartLine: 25},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 4}, {StartLine: 14}, {StartLine: 17},
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
