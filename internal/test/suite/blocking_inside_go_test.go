package suite

import (
	"testing"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestBlockingInsideGo(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "blocking_inside_go.clj",
			RuleID:        "blocking-inside-go",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Blocking function detected within the GO block go", StartLine: 4},
				{Message: "Blocking function detected within the GO block go", StartLine: 8},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
