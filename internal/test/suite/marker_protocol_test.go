package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMarkerProtocol(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "marker_protocol.clj",
		RuleID:        "marker-protocol",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "Marker protocol", StartLine: 3, Severity: rules.SeverityHint},
			{Message: "Marker protocol", StartLine: 5, Severity: rules.SeverityHint},
		},
		ForbiddenFindings: []framework.ExpectedFinding{
			{StartLine: 8},
			{StartLine: 11},
			{StartLine: 15},
		},
	})
}
