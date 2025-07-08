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
                {Message:   "Function 'greet-user' performs side effects", StartLine: 4},
				{Message: "Function 'accumulate' performs side effects", StartLine: 17},
				{Message: "Function 'side-effect-check' appears to be pure", StartLine: 26},
				{Message: "Function 'lazy-numbers' performs", StartLine: 34},
				{Message: "Function 'check-even' appears to be pure", StartLine: 39},
				{Message: "Function 'show-and-scale' performs", StartLine: 48},
				{Message: "Function 'track-and-inc' performs", StartLine: 59},
				{Message: "Function 'lazy-show-nums' performs", StartLine: 69},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
				framework.RunRuleTest(t, tc)
        })
    }
}
