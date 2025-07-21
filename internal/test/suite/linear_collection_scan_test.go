package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestLinearCollectionScan(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "linear_collection_scan.clj",
			RuleID:        "linear-collection-scan",
			ExpectedFindings: []framework.ExpectedFinding{

				{Message: "Manual loop for finding elements", StartLine: 12},

				{Message: "Counting filtered results can be done in single pass", StartLine: 25},

				{Message: "Using sort for min/max detected", StartLine: 34},

				{Message: "Counting filtered results", StartLine: 43},

				{Message: "Counting filtered results", StartLine: 43},

				{Message: "Multiple nested map/filter detected", StartLine: 52},

				{Message: "Multiple nested map/filter detected", StartLine: 52},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}

func TestLinearCollectionScanDebug(t *testing.T) {
	testCase := framework.RuleTestCase{
		FileToAnalyze:    "linear_collection_scan.clj",
		RuleID:           "linear-collection-scan",
		ExpectedFindings: []framework.ExpectedFinding{},
	}

	framework.DebugRuleTest(t, testCase)
}
