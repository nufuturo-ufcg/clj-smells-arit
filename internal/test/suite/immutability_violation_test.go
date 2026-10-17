package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestImmutabilityViolation(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "immutability_violation.clj",
			RuleID:        "immutability-violation",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Found `def` inside a local scope", StartLine: 8, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Found `defonce` inside a local scope", StartLine: 35, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Found `def` inside a local scope", StartLine: 41, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Found `def` inside a local scope", StartLine: 54, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Found alter-var-root", StartLine: 13, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Found `ref-set` outside of `dosync`", StartLine: 128, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "type-hinted Java array", StartLine: 141, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "type-hinted Java array", StartLine: 152, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "type-hinted Java array", StartLine: 160, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 18}, {StartLine: 22}, {StartLine: 26}, {StartLine: 30}, {StartLine: 46}, {StartLine: 146}, {StartLine: 156},
				{StartLine: 119},
				{StartLine: 123},
				{StartLine: 131},
				{StartLine: 135},
			},
		},
		{
			FileToAnalyze: "immutability_violation_alter_var_root.clj",
			RuleID:        "immutability-violation",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Found alter-var-root", StartLine: 6, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 9},
				{StartLine: 12},
				{StartLine: 15},
				{StartLine: 18},
				{StartLine: 21},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
