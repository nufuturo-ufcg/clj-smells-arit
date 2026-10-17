package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

// Method implementations inside reify, proxy, and deftype name interface
// methods and bind their parameters; neither may resolve to clojure.core.
func TestMethodImplResolution(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "method_impl_resolution.clj",
		RuleID:        "monolithic-namespace-split",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "Use of load", StartLine: 14, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 7}, {StartLine: 10}},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:    "method_impl_resolution.clj",
		RuleID:           "unnecessary-laziness",
		AssertNoFindings: true,
	})
}
