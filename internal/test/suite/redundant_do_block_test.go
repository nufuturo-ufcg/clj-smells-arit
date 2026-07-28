package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestRedundantDoBlock(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "redundant_do_block.clj",
			RuleID:        "redundant-do-block",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Redundant `do` block found. The surrounding `let` form", StartLine: 15},
				{Message: "Redundant `do` block found. The surrounding `when` form", StartLine: 22},
				{Message: "Redundant `do` block with a single expression found within `if`", StartLine: 30},
				{Message: "Redundant `do` block with a single expression found within `if`", StartLine: 32},
				{Message: "Redundant `do` block with a single expression found within `defn`", StartLine: 37},
				{Message: "Redundant `do` block found. The surrounding `try` form", StartLine: 43},
				{Message: "Redundant `do` block found. The surrounding `catch` form", StartLine: 47},
				{Message: "Redundant `do` block found. The surrounding `doseq` form", StartLine: 54},
				{Message: "Redundant `do` block with a single expression found within `if`", StartLine: 62},
				{Message: "Redundant `do` block with a single expression found within `if`", StartLine: 64},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
