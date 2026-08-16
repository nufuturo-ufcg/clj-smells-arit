package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestImproperEmptinessCheck(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "improper_emptiness_check.clj",
		RuleID:        "improper-emptiness-check",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "(empty? xs)", StartLine: 4},
			{Message: "(seq xs)", StartLine: 7},
			{Message: "(seq xs)", StartLine: 12},
			{Message: "(seq xs)", StartLine: 15},
			{Message: "(seq xs)", StartLine: 21},
			{Message: "(when (seq xs)", StartLine: 24},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 28},
		},
	})
}
