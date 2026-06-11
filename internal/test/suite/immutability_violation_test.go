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
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Found `def` inside a local scope", StartLine: 8},
				{Message: "Found state mutation function call: `alter-var-root`", StartLine: 13},
				{Message: "Found `reset!`", StartLine: 18},
				{Message: "Found state mutation function call: `set!`", StartLine: 22},
				{Message: "Found state mutation function call: `intern`", StartLine: 26},
				{Message: "Found `def` inside a local scope", StartLine: 30},
				{Message: "Found `defonce` inside a local scope", StartLine: 35},
				{Message: "Found `def` inside a local scope", StartLine: 41},
				{Message: "Found state mutation function call: `aset`", StartLine: 46},
				{Message: "Found `def` inside a local scope", StartLine: 54},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
				framework.RunRuleTest(t, tc)
        })
    }
}
