package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestRefsInDependencyVector(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "refs_in_dependency_vector.clj",
		RuleID:        "refs-in-dependency-vector",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "used directly in an effect dependency vector", StartLine: 5},
			{Message: "used directly in an effect dependency vector", StartLine: 9},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 12}, {StartLine: 15}},
	})
}
