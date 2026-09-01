package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestImproperEmptinessCheck(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "improper_emptiness_check.clj",
		RuleID:        "improper-emptiness-check",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "(empty? xs)", StartLine: 4, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "(seq xs)", StartLine: 7, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "(seq xs)", StartLine: 12},
			{Message: "(seq xs)", StartLine: 15, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "return contract is unknown", StartLine: 21, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "(when (seq xs)", StartLine: 24, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "(empty? [...])", StartLine: 32, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "(seq [...])", StartLine: 35, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "(seq (keys ...))", StartLine: 39, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "(seq (vals ...))", StartLine: 43, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "(seq (map ...))", StartLine: 47, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "(seq (external-items ...))", StartLine: 51, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "(seq nil)", StartLine: 81, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 28},
			{StartLine: 55},
			{StartLine: 60},
			{StartLine: 65},
			{StartLine: 70},
			{StartLine: 74},
			{StartLine: 77},
		},
	})

	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "improper_emptiness_check.cljs",
		RuleID:        "improper-emptiness-check",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "(when (seq (.-value ...))", StartLine: 6, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "return contract is unknown", StartLine: 11, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "(seq (.-value ...))", StartLine: 16, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
	})
}
