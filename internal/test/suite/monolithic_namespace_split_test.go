package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMonolithicNamespaceSplit(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "monolithic_namespace_split.clj",
			RuleID:        "monolithic-namespace-split",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Use of load stitches", StartLine: 6},
				{Message: "Use of in-ns switches", StartLine: 12},
				{Message: "Use of load stitches", StartLine: 17},
				{Message: "Use of in-ns switches", StartLine: 18},
				{Message: "Use of load stitches", StartLine: 22},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
