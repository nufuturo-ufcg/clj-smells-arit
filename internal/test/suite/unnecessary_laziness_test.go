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
			{Message: "Proven redundant laziness", StartLine: 4, RequireConfidence: rules.ConfidenceProven},
			{Message: "immediately materialized by `vec`", StartLine: 7},
			{Message: "immediately materialized by `vec`", StartLine: 10},
			{Message: "immediately materialized by `vec`", StartLine: 13},
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
			{StartLine: 58},
		},
	})

	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "unnecessary_laziness_non_evaluated.clj",
		RuleID:            "unnecessary-laziness",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 4}, {StartLine: 7}, {StartLine: 9}, {StartLine: 12}},
	})
}
