package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMisuseOfDynamicScope(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "misuse_of_dynamic_scope.clj",
		RuleID:        "misuse-of-dynamic-scope",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "Defining custom dynamic variable", StartLine: 12},
			{Message: "Binding custom dynamic variable", StartLine: 30, Severity: rules.SeverityHint},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 3}, {StartLine: 4},
			{StartLine: 8}, {StartLine: 15}, {StartLine: 24},
		},
	})
}
