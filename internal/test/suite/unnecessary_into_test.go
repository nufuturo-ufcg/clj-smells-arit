package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnnecessaryInto(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "unnecessary_into.clj",
		RuleID:        "unnecessary-into",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "lazy `map` result", StartLine: 5, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "lazy `filter` result", StartLine: 8, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "lazy `take` result", StartLine: 11, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "lazy `distinct` result", StartLine: 14, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "lazy `map` result", StartLine: 18, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 22},
			{StartLine: 25},
			{StartLine: 28},
			{StartLine: 32},
			{StartLine: 35},
			{StartLine: 39},
			{StartLine: 42},
			{StartLine: 46},
			{StartLine: 50},
			{StartLine: 54},
			{StartLine: 58},
			{StartLine: 61},
			{StartLine: 65},
			{StartLine: 68},
			{StartLine: 72},
		},
	})
}
