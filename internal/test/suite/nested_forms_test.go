package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestNestedForms(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "nested_forms.clj",
			RuleID:        "nested-forms",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Safe nested let flattening detected (3 consecutive forms)", StartLine: 4, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Safe nested doseq flattening detected (2 consecutive forms)", StartLine: 10, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Safe nested doseq flattening detected (2 consecutive forms)", StartLine: 15, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			},
		},
		{
			FileToAnalyze:    "nested_forms_precision.clj",
			RuleID:           "nested-forms",
			ExpectedFindings: nil,
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 5},
				{StartLine: 11},
				{StartLine: 19},
				{StartLine: 26},
				{StartLine: 33},
				{StartLine: 40},
				{StartLine: 49},
				{StartLine: 55},
				{StartLine: 66},
				{StartLine: 73},
				{StartLine: 80},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
