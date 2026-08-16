package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestOverengineeringCoreAsync(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "overengineering_core_async.clj",
		RuleID:        "overengineering-with-core-async",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "used only to return one value", StartLine: 5},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 10}, {StartLine: 15}},
	})
}
