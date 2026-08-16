package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnnecessaryMacros(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "unnecessary_macros.clj",
		RuleID:        "unnecessary-macros",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "prefer a normal function", StartLine: 3},
			{Message: "prefer a normal function", StartLine: 6},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 9}, {StartLine: 12}, {StartLine: 16}},
	})
}
