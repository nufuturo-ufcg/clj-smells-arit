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
                {Message:   "Function 'greet-user' performs side effects (I/O operations) without explicit indication. Consider adding '!' suffix or using 'doseq' for side-effect operations to make the impurity explicit.", StartLine: 5},
				//{Message: "", StartLine: 17},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.DebugRuleTest(t, tc)
        })
    }
}
