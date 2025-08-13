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

				{Message: "Redundant `do` block found. The surrounding `let` form", StartLine: 15},            // Example 1 - let body
				{Message: "Redundant `do` block found. The surrounding `when` form", StartLine: 22},           // Example 2 - when body
				{Message: "Redundant `do` block found. The surrounding `if` form", StartLine: 30},             // Example 3 - if then branch
				{Message: "Redundant `do` block found. The surrounding `if` form", StartLine: 33},             // Example 3 - if else branch
				{Message: "Redundant `do` block with a single expression found within `defn`", StartLine: 39}, // Example 4 - defn single expression
				{Message: "Redundant `do` block found. The surrounding `try` form", StartLine: 45},            // Example 5 - try body
				{Message: "Redundant `do` block found. The surrounding `catch` form", StartLine: 49},          // Example 5 - catch body
				{Message: "Redundant `do` block found. The surrounding `doseq` form", StartLine: 56},          // Example 6 - doseq body
				{Message: "Redundant `do` block found. The surrounding `if` form", StartLine: 64},             // Example 7 - loop if then
				{Message: "Redundant `do` block found. The surrounding `if` form", StartLine: 67},             // Example 7 - loop if else
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
