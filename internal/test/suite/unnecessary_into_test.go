package suite

import (
	"testing"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnnecessaryInto(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "unnecessary_into.clj",
			RuleID:        "unnecessary-into",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "vec coll", StartLine: 4},
				{Message: "set coll", StartLine: 7},
				{Message: "into {} (map", StartLine: 10},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
