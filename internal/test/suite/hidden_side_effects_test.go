package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestHiddenSideEffects(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "hidden_side_effects.clj",
            RuleID:        "hidden-side-effects",  // ID da sua regra
            ExpectedFindings: []framework.ExpectedFinding{
                {Message:   "Function 'greet-user' performs side effects", StartLine: 5},
				{Message: "Function 'side-effect-check' appears to be pure", StartLine: 27},
				{Message: "Function 'lazy-numbers' performs", StartLine: 35},
				{Message: "Function 'check-even' appears to be pure", StartLine: 40},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}
