package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestNestedAtoms(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "nested_atoms.clj",
			RuleID:        "nested-atoms",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 3},
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 8},
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 18},
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 19},
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 20},
				{Message: "being inserted into another stateful reference", StartLine: 37},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 24},
				{StartLine: 31},
				{StartLine: 42},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
