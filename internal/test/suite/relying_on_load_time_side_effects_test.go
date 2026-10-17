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
			{Message: "runs while the namespace is loaded", StartLine: 3, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "runs while the namespace is loaded", StartLine: 4, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "runs while the namespace is loaded", StartLine: 13, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "runs while the namespace is loaded", StartLine: 18, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "runs while the namespace is loaded", StartLine: 50, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "runs while the namespace is loaded", StartLine: 51, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
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
	// Only the :bb branch reads the file at load time; on :clj the same name
	// is a macro evaluated at compile time, so no proven effect exists.
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "relying_on_load_time_platform.cljc",
		RuleID:            "relying-on-load-time-side-effects",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 6}},
	})
}
