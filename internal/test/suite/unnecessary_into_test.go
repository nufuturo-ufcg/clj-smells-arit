package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnnecessaryInto(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "unnecessary_into.clj",
		RuleID:        "unnecessary-into",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "lazy `map` result", StartLine: 5},
			{Message: "lazy `filter` result", StartLine: 8},
			{Message: "lazy `take` result", StartLine: 11},
			{Message: "lazy `distinct` result", StartLine: 14},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 18},
			{StartLine: 21},
			{StartLine: 24},
			{StartLine: 28},
			{StartLine: 31},
			{StartLine: 35},
			{StartLine: 38},
			{StartLine: 42},
			{StartLine: 46},
			{StartLine: 50},
			{StartLine: 54},
			{StartLine: 57},
			{StartLine: 61},
			{StartLine: 64},
			{StartLine: 68},
		},
	})
}
