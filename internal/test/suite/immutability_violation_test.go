package suite

import (
	"github.com/thlaurentino/arit/internal/test/framework"
	"testing"
)

func TestImmutabilityViolation(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "immutability_violation.clj",
			RuleID:        "immutability-violation",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Found `def` inside a local scope", StartLine: 8},
				{Message: "Found `def` inside a local scope", StartLine: 30},
				{Message: "Found `defonce` inside a local scope", StartLine: 35},
				{Message: "Found `def` inside a local scope", StartLine: 41},
				{Message: "Found `def` inside a local scope", StartLine: 54},
				{Message: "Found `ref-set` outside of `dosync`", StartLine: 128},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 119},
				{StartLine: 123},
				{StartLine: 131},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
