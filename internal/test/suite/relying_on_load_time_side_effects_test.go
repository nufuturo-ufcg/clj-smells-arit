package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestRelyingOnLoadTimeSideEffects(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "relying_on_load_time_side_effects.clj",
		RuleID:        "relying-on-load-time-side-effects",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "runs while the namespace is loaded", StartLine: 3},
			{Message: "runs while the namespace is loaded", StartLine: 4},
			{Message: "runs while the namespace is loaded", StartLine: 15},
			{Message: "runs while the namespace is loaded", StartLine: 42},
			{Message: "runs while the namespace is loaded", StartLine: 43},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 6}, {StartLine: 7}, {StartLine: 10},
			{StartLine: 20}, {StartLine: 28}, {StartLine: 32}, {StartLine: 36},
			{StartLine: 39},
		},
	})
}
