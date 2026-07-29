package suite

import (
	"testing"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestThreadIgnorance(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "thread_ignorance.clj",
			RuleID:        "thread-ignorance",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "depth 3", StartLine: 4},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
