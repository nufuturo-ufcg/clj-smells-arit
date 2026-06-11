package suite

import (
	"testing"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestProductionDoall(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "production_doall.clj",
			RuleID:        "production-doall",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "mapv, into, vec", StartLine: 4},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
