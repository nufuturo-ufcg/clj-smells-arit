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
			{Message: "Defining custom dynamic variable", StartLine: 12, RequireContextual: true},
			{Message: "read across an asynchronous or lazy boundary", StartLine: 9, RequireContextual: true},
			{Message: "read across an asynchronous or lazy boundary", StartLine: 16, RequireContextual: true},
			{Message: "Binding custom dynamic variable", StartLine: 30, Severity: rules.SeverityHint, RequireContextual: true},
			{Message: "read across an asynchronous or lazy boundary", StartLine: 31, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 3}, {StartLine: 4},
			{StartLine: 8}, {StartLine: 15}, {StartLine: 24},
		},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "misuse_of_dynamic_scope_explicit.clj",
		RuleID:        "misuse-of-dynamic-scope",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "read across an asynchronous or lazy boundary", StartLine: 6, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 9}},
	})
}
