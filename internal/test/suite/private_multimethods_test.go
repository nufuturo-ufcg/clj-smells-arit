package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestPrivateMultimethods(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "private_multimethods.clj",
			RuleID:        "private-multimethods",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Private multimethod", StartLine: 6, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Private multimethod", StartLine: 15, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Private multimethod", StartLine: 29, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Private multimethod", StartLine: 37, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Private multimethod", StartLine: 46, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Private multimethod", StartLine: 59, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Private multimethod", StartLine: 67, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 78},
				{StartLine: 81},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "private_multimethods_invalid.clj",
		RuleID:            "private-multimethods",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 4}},
	})
}
