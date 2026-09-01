package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestNonIdiomaticParameterBinding(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "non_idiomatic_parameter_binding.clj",
			RuleID:        "non-idiomatic-parameter-binding",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Non-idiomatic parameter binding", StartLine: 3, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Non-idiomatic parameter binding", StartLine: 6, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Non-idiomatic parameter binding", StartLine: 9, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Non-idiomatic parameter binding", StartLine: 16, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 12},
				{StartLine: 20},
				{StartLine: 24},
				{StartLine: 28},
			},
		},
		{
			FileToAnalyze: "non_idiomatic_parameter_binding_multi_arity.clj",
			RuleID:        "non-idiomatic-parameter-binding",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Non-idiomatic parameter binding", StartLine: 3, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Non-idiomatic parameter binding", StartLine: 16, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 8},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
