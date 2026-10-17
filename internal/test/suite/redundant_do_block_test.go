package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestRedundantDoBlock(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "redundant_do_block.clj",
			RuleID:        "redundant-do-block",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Redundant `do` block found. The surrounding `let` form", StartLine: 15, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block found. The surrounding `when` form", StartLine: 22, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block with a single expression found within `if`", StartLine: 30, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block with a single expression found within `if`", StartLine: 32, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block with a single expression found within `defn`", StartLine: 37, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block found. The surrounding `try` form", StartLine: 43, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block found. The surrounding `catch` form", StartLine: 47, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block found. The surrounding `doseq` form", StartLine: 54, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block with a single expression found within `if`", StartLine: 62, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block with a single expression found within `if`", StartLine: 64, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block with a single expression found within `case`", StartLine: 167, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant `do` block with a single expression found within `case`", StartLine: 171, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Redundant empty `do` block found within `cond`. Replace it with `nil`", StartLine: 176, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 88},
				{StartLine: 91},
				{StartLine: 158},
				{StartLine: 165},
				{StartLine: 172},
				{StartLine: 182},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
