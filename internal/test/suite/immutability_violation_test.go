package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestImmutabilityViolation(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "immutability_violation.clj",
            RuleID:        "immutability-violation",  

            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
				framework.RunRuleTest(t, tc)
        })
    }
}
