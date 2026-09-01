package suite

import (
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
	"testing"
)

func TestSingleSegmentNamespace(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze:     "single_segment_namespace.clj",
			RuleID:            "single-segment-namespace",
			ExpectedFindings:  []framework.ExpectedFinding{{StartLine: 16, Severity: rules.SeverityHint}},
			ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 11}, {StartLine: 14}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
