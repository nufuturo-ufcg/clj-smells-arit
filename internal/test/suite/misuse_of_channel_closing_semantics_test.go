package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMisuseOfChannelClosingSemantics(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "misuse_of_channel_closing_semantics.clj",
			RuleID:        "misuse-of-channel-closing-semantics",
			ExpectedFindings: []framework.ExpectedFinding{
				// Put: sentinel value (all use the message "Sentinel value ... in ...")
				{Message: "Sentinel value", StartLine: 8, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Comparison with sentinel", StartLine: 14, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Sentinel value", StartLine: 19, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Sentinel value", StartLine: 23, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Sentinel value", StartLine: 27, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Sentinel value", StartLine: 31, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Sentinel value", StartLine: 35, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Sentinel value", StartLine: 39, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Sentinel value", StartLine: 42, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				// Comparison: (= or not= sentinel take-form)
				{Message: "Comparison with sentinel", StartLine: 45, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Comparison with sentinel", StartLine: 46, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				// More sentinels (stem-based)
				{Message: "Sentinel value", StartLine: 49, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
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
		FileToAnalyze:     "misuse_of_channel_closing_semantics_precision.clj",
		RuleID:            "misuse-of-channel-closing-semantics",
		// Zero-FP policy: a single :done on a channel that carries no data in
		// the file is a control signal, not a misused sentinel (line 22).
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 6}, {StartLine: 10}, {StartLine: 14}, {StartLine: 18}, {StartLine: 22}},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "channel_protocol_signature.clj",
		RuleID:            "misuse-of-channel-closing-semantics",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 4}},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "misuse_of_channel_closing_semantics_invalid.clj",
		RuleID:            "misuse-of-channel-closing-semantics",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 5}, {StartLine: 6}, {StartLine: 7}},
	})
}
