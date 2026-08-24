package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMapWithNilValues(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "map_with_nil_values.clj",
		RuleID:        "map-with-nil-values",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "Explicit 'nil' value associated with a key in a map literal", StartLine: 7, Severity: rules.SeverityHint},
			{Message: "Explicit 'nil' value associated with a key using 'assoc'", StartLine: 11, Severity: rules.SeverityHint},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 3},
		},
	})
}
