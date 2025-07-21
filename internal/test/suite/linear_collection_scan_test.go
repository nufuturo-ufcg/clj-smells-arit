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

				{Message: "Manual loop for finding elements can be replaced with 'some' or 'filter'", StartLine: 12},

				{Message: "Counting filtered results can be done in single pass", StartLine: 25},

				{Message: "Using sort for min/max detected. Prefer 'reduce' or 'apply min/max' for efficiency", StartLine: 34},

				{Message: "Using count/filter for existence check detected. Prefer 'some' or 'not-any?' for efficiency", StartLine: 43},

				{Message: "Multiple nested map/filter detected. Prefer function composition or threading macros to avoid multiple collection passes", StartLine: 52},
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
