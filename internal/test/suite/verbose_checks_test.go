package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestVerboseChecks(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "verbose_checks.clj",
			RuleID:        "verbose-checks",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "= 0 x", StartLine: 4, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "= x true", StartLine: 6, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "= nil x", StartLine: 8, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "+ 1 x", StartLine: 10, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: ">= x 0", StartLine: 14, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "= 0 (long ...)", StartLine: 25, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "not= nil x", StartLine: 29, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "(.indexOf ...)", StartLine: 33, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "(.length ...)", StartLine: 36, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "(.size ...)", StartLine: 39, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "(.indexOf ...)", StartLine: 42, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "(.size ...)", StartLine: 45, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "(.size ...)", StartLine: 48, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "(.size ...)", StartLine: 51, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Double/NaN", StartLine: 85, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "(+ ...)", StartLine: 88, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "= 0 x", StartLine: 91, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 17},
				{StartLine: 21},
				{StartLine: 55},
				{StartLine: 58},
				{StartLine: 62},
				{StartLine: 65},
				{StartLine: 68},
				{StartLine: 71},
				{StartLine: 74},
				{StartLine: 78},
				{StartLine: 81},
				{StartLine: 95},
			},
		},
		{
			FileToAnalyze:     "verbose_checks_assertions.clj",
			RuleID:            "verbose-checks",
			ExpectedFindings:  []framework.ExpectedFinding{},
			ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 6}, {StartLine: 7}, {StartLine: 8}},
		},
		{
			FileToAnalyze: "verbose_checks.cljs",
			RuleID:        "verbose-checks",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "= value true", StartLine: 5, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "= value 0", StartLine: 9, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Verbose boolean if", StartLine: 16, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 13}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
