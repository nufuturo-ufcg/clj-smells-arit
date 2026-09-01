package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestRefsInDependencyVector(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "refs_in_dependency_vector.clj",
		RuleID:        "refs-in-dependency-vector",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "used directly in an effect dependency vector", StartLine: 9},
			{Message: "used directly in an effect dependency vector", StartLine: 37},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 5},
			{StartLine: 14},
			{StartLine: 17},
			{StartLine: 20},
			{StartLine: 23},
			{StartLine: 28},
			{StartLine: 33},
		},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "refs_in_dependency_vector_binding_forms.clj",
		RuleID:        "refs-in-dependency-vector",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "used directly in an effect dependency vector", StartLine: 6, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "used directly in an effect dependency vector", StartLine: 10, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "used directly in an effect dependency vector", StartLine: 15, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 20},
			{StartLine: 25},
			{StartLine: 29},
			{StartLine: 33},
			{StartLine: 38},
			{StartLine: 43},
		},
	})
}
