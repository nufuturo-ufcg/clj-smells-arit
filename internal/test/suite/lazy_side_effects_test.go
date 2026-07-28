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
				{Message: "passed to lazy function 'map' may contain side effects", StartLine: 7},
				{Message: "passed to lazy function 'filter' may contain side effects", StartLine: 11},
				{Message: "passed to lazy function 'for' may contain side effects", StartLine: 15},
				{Message: "passed to lazy function 'map' may contain side effects", StartLine: 20},
				{Message: "passed to lazy function 'filter' may contain side effects", StartLine: 24},
				{Message: "passed to lazy function 'map' may contain side effects", StartLine: 29},
				{Message: "passed to lazy function 'map' may contain side effects", StartLine: 33},
				{Message: "passed to lazy function 'map' may contain side effects", StartLine: 38},
				{Message: "passed to lazy function 'map' may contain side effects", StartLine: 42},
				{Message: "passed to lazy function 'for' may contain side effects", StartLine: 46},
				{Message: "passed to lazy function 'map' may contain side effects", StartLine: 50},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
				framework.RunRuleTest(t, tc)
        })
    }
}
