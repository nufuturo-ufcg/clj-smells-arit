package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestExcessiveRefers(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "excessive_refers.clj",
			RuleID:        "excessive-refers",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "explicitly refers 24 Vars", StartLine: 136},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 27},
				{StartLine: 130},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
