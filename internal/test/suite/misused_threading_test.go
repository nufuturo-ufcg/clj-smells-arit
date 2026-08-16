package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMisusedThreading(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "misused_threading.clj",
		RuleID:        "misused-threading",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "3 resolved pipeline steps consistently", StartLine: 6},
			{Message: "2 resolved pipeline steps consistently", StartLine: 12},
			{Message: "2 resolved pipeline steps consistently", StartLine: 17},
			{Message: "2 resolved pipeline steps consistently", StartLine: 22},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 28},
			{StartLine: 31},
			{StartLine: 35},
			{StartLine: 41},
			{StartLine: 45},
			{StartLine: 49},
			{StartLine: 53},
			{StartLine: 57},
			{StartLine: 63},
			{StartLine: 67},
			{StartLine: 71},
			{StartLine: 75},
		},
	})
}
