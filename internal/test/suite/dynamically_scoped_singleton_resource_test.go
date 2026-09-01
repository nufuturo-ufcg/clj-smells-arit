package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestDynamicallyScopedSingletonResource(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "dynamically_scoped_singleton_resource.clj",
		RuleID:        "dynamically-scoped-singleton-resource",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "Passing dynamic variable `*current-conn*`", StartLine: 8, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 11},
			{StartLine: 14},
			{StartLine: 17},
		},
	})
}
