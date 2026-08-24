package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMisuseOfChannelClosingSemantics(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "misuse_of_channel_closing_semantics.clj",
			RuleID:        "misuse-of-channel-closing-semantics",
			ExpectedFindings: []framework.ExpectedFinding{
				// Put: sentinel value (todas usam mensagem "Sentinel value ... in ...")
				{Message: "Sentinel value", StartLine: 8},
				{Message: "Comparison with sentinel", StartLine: 14},
				{Message: "Sentinel value", StartLine: 19},
				{Message: "Sentinel value", StartLine: 23},
				{Message: "Sentinel value", StartLine: 27},
				{Message: "Sentinel value", StartLine: 31},
				{Message: "Sentinel value", StartLine: 35},
				{Message: "Sentinel value", StartLine: 39},
				{Message: "Sentinel value", StartLine: 42},
				// Comparison: (= ou not= sentinel take-form)
				{Message: "Comparison with sentinel", StartLine: 45},
				{Message: "Comparison with sentinel", StartLine: 46},
				// More sentinels (stem-based)
				{Message: "Sentinel value", StartLine: 49},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 50}, {StartLine: 51}, {StartLine: 52}, {StartLine: 53}, {StartLine: 54},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "channel_protocol_signature.clj",
		RuleID:            "misuse-of-channel-closing-semantics",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 4}},
	})
}
