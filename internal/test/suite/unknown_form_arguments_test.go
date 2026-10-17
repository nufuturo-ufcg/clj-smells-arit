package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

// Arguments of a form that may be a macro of another namespace are not proven
// to run, and a tagged literal is not a map literal.
func TestUnknownFormArguments(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "unknown_form_arguments.clj",
		RuleID:        "immutability-violation",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "Found `def`", StartLine: 12, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 7}},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:    "unknown_form_arguments.clj",
		RuleID:           "implicit-namespace-dependencies",
		AssertNoFindings: true,
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:    "unknown_form_arguments.clj",
		RuleID:           "redundant-do-block",
		AssertNoFindings: true,
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:    "unknown_form_arguments.clj",
		RuleID:           "map-with-nil-values",
		AssertNoFindings: true,
	})
}
