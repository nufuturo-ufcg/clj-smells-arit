package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestProductionDoall(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "production_doall.clj",
		RuleID:        "production-doall",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "Redundant `doall` around `mapv`", StartLine: 5},
			{Message: "Redundant `doall` around `filterv`", StartLine: 8},
			{Message: "forces realization", StartLine: 12},
			{Message: "forces realization", StartLine: 15},
			{Message: "lifecycle boundary", StartLine: 20},
			{Message: "forces realization", StartLine: 24},
			{Message: "forces realization", StartLine: 27},
			{Message: "forces realization", StartLine: 32},
			{Message: "forces realization", StartLine: 39},
			{Message: "forces realization", StartLine: 43},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 36},
			{StartLine: 47},
			{StartLine: 50},
			{StartLine: 54},
			{StartLine: 57},
		},
	})
}
