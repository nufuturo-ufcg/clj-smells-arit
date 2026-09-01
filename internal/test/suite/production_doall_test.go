package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestProductionDoall(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "production_doall.clj",
		RuleID:        "production-doall",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "Redundant `doall` around `mapv`", StartLine: 5, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "Redundant `doall` around `filterv`", StartLine: 8, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "Redundant `doall` around `vec`", StartLine: 11, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "Redundant `doall` around `into`", StartLine: 14, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			{Message: "forces realization", StartLine: 18, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "forces realization", StartLine: 21},
			{Message: "lifecycle boundary", StartLine: 26},
			{Message: "forces realization", StartLine: 30},
			{Message: "forces realization", StartLine: 33},
			{Message: "forces realization", StartLine: 38},
			{Message: "forces realization", StartLine: 45},
			{Message: "forces realization", StartLine: 49},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 42},
			{StartLine: 53},
			{StartLine: 56},
			{StartLine: 60},
			{StartLine: 63},
		},
	})

	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "production_doall_invalid.clj",
		RuleID:            "production-doall",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "forces realization", StartLine: 8, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "forces realization", StartLine: 11, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			{Message: "forces realization", StartLine: 14, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 5}},
	})
}
