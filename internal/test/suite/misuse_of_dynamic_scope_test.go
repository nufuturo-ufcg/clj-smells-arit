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
			{Message: "Defining custom dynamic variable", StartLine: 12, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "Binding custom dynamic variable", StartLine: 30, Severity: rules.SeverityHint, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "read across an asynchronous or lazy boundary", StartLine: 31, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 3}, {StartLine: 4},
			{StartLine: 8}, {StartLine: 15}, {StartLine: 24},
			// Zero-FP policy: future conveys bindings, and these reads happen
			// inside a binding established in the same worker.
			{StartLine: 9}, {StartLine: 16},
		},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "misuse_of_dynamic_scope_explicit.clj",
		RuleID:        "misuse-of-dynamic-scope",
		// Zero-FP policy: future conveys the caller's bindings, so the read at
		// line 6 sees the active binding.
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 6}, {StartLine: 9}},
	})
}
