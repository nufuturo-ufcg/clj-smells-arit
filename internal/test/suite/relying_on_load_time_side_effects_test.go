package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestRelyingOnLoadTimeSideEffects(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "relying_on_load_time_side_effects.clj",
		RuleID:        "relying-on-load-time-side-effects",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "runs while the namespace is loaded", StartLine: 3},
			{Message: "runs while the namespace is loaded", StartLine: 4},
			{Message: "runs while the namespace is loaded", StartLine: 13, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "runs while the namespace is loaded", StartLine: 18},
			{Message: "runs while the namespace is loaded", StartLine: 50},
			{Message: "runs while the namespace is loaded", StartLine: 51},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 6}, {StartLine: 7}, {StartLine: 10},
			{StartLine: 23}, {StartLine: 30}, {StartLine: 34}, {StartLine: 38},
			{StartLine: 42}, {StartLine: 47},
		},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "relying_on_load_time_static_resource.clj",
		RuleID:            "relying-on-load-time-side-effects",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 3}},
	})
}
