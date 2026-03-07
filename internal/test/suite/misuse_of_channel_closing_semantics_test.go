package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMisuseOfChannelClosingSemantics(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "misuse_of_channel_closing_semantics.clj",
			RuleID:       "misuse-of-channel-closing-semantics",
			ExpectedFindings: []framework.ExpectedFinding{
				// Sentinel :done — put! and comparison
				{Message: "put!", StartLine: 8},
				{Message: "Comparison with sentinel", StartLine: 14},
				// Sentinel :EOF
				{Message: "put!", StartLine: 19},
				{Message: "Comparison with sentinel", StartLine: 20},
				// Sentinel :end
				{Message: "put!", StartLine: 23},
				{Message: "Comparison with sentinel", StartLine: 24},
				// Sentinel ::end
				{Message: "put!", StartLine: 27},
				{Message: "Comparison with sentinel", StartLine: 28},
				// Sentinel :eof
				{Message: "put!", StartLine: 31},
				{Message: "Comparison with sentinel", StartLine: 32},
				// Sentinel :stream/done (namespaced)
				{Message: "put!", StartLine: 35},
				{Message: "Comparison with sentinel", StartLine: 36},
				// Sentinel with >! (parking put)
				{Message: "put!", StartLine: 39},
				// Sentinel with >!! (blocking put)
				{Message: "put!", StartLine: 42},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
