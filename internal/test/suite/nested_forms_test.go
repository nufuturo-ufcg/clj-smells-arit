package suite

import (
	"testing"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestNestedForms(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "nested_forms.clj",
			RuleID:        "nested-forms",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "depth: 2, forms: let → let", StartLine: 4},
				{Message: "depth: 2, forms: doseq → doseq", StartLine: 9},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
