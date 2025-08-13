package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestLazySideEffects(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "lazy_side_effects.clj",
            RuleID:        "lazy-side-effects", 
			ExpectedFindings: []framework.ExpectedFinding{
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
				framework.DebugRuleTest(t, tc)
        })
    }
}
