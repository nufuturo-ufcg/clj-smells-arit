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
			{Message: "used only to return one value", StartLine: 6},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 11}, {StartLine: 16}, {StartLine: 22}, {StartLine: 28}, {StartLine: 34}, {StartLine: 40}},
	})
}
