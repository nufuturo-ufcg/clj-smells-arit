package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestExplicitRecursion(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "explicit_recursion.clj",
			RuleID:        "explicit-recursion",
			ExpectedFindings: []framework.ExpectedFinding{
				// MAP patterns
				{Message: "transformation (map", StartLine: 6},  // double-nums-recursive
				{Message: "transformation (map", StartLine: 28}, // square-and-add-one
				{Message: "transformation (map", StartLine: 49}, // uppercase-strings
				{Message: "transformation (map", StartLine: 72}, // strings-to-ints

				// REDUCE patterns
				{Message: "accumulator (reduce", StartLine: 12}, // sum-list
				{Message: "accumulator (reduce", StartLine: 35}, // product-list
				{Message: "accumulator (reduce", StartLine: 56}, // concat-all
				{Message: "accumulator (reduce", StartLine: 79}, // all-true?

				// FILTER patterns
				{Message: "filtering pattern", StartLine: 18}, // get-even-numbers
				{Message: "filtering pattern", StartLine: 41}, // get-positive
				{Message: "filtering pattern", StartLine: 62}, // get-long-words
				{Message: "filtering pattern", StartLine: 85}, // get-valid-numbers
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
