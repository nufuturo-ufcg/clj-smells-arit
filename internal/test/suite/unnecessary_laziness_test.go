package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnnecessaryLaziness(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "unnecessary_laziness.clj",
		RuleID:        "unnecessary-laziness",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "immediately materialized by `vec`", StartLine: 4},
			{Message: "immediately materialized by `vec`", StartLine: 7},
			{Message: "immediately materialized by `vec`", StartLine: 10},
			{Message: "immediately materialized by `vec`", StartLine: 13},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 19},
			{StartLine: 22},
			{StartLine: 25},
			{StartLine: 28},
			{StartLine: 30},
			{StartLine: 33},
			{StartLine: 36},
		},
	})
}
