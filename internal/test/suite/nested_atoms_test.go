package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestNestedAtoms(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "nested_atoms.clj",
			RuleID:        "nested-atoms",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 3, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 8, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 18, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 19, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Found nested Atom/Ref/Volatile", StartLine: 20, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "being inserted into another stateful reference", StartLine: 37, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "being created and inserted into another stateful reference", StartLine: 42, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 24},
				{StartLine: 31},
				{StartLine: 46},
				{StartLine: 50},
			},
		},
		{
			FileToAnalyze: "nested_atoms_invalid_arity.clj",
			RuleID:        "nested-atoms",
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 4},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
