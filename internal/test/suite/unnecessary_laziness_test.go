package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnnecessaryLaziness(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "unnecessary_laziness.clj",
		RuleID:        "unnecessary-laziness",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "Proven redundant laziness", StartLine: 4, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "immediately materialized by `vec`", StartLine: 7, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "immediately materialized by `vec`", StartLine: 10, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "immediately materialized by `vec`", StartLine: 13, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "Proven redundant laziness", StartLine: 58, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "Proven redundant laziness", StartLine: 59, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "Proven redundant laziness", StartLine: 60, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "Proven redundant laziness", StartLine: 61, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "immediately materialized by `vec`", StartLine: 65, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "immediately materialized by `vec`", StartLine: 66, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "immediately materialized by `vec`", StartLine: 67, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "immediately materialized by `vec`", StartLine: 68, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "immediately materialized by `vec`", StartLine: 72, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "immediately materialized by `vec`", StartLine: 76, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 16},
			{StartLine: 22},
			{StartLine: 25},
			{StartLine: 28},
			{StartLine: 31},
			{StartLine: 34},
			{StartLine: 37},
			{StartLine: 40},
			{StartLine: 43},
			{StartLine: 46},
			{StartLine: 49},
			{StartLine: 52},
			{StartLine: 55},
		},
	})

	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "unnecessary_laziness_non_evaluated.clj",
		RuleID:            "unnecessary-laziness",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 4}, {StartLine: 7}, {StartLine: 9}, {StartLine: 12}},
	})
}
