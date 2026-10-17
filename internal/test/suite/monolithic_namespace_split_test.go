package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMonolithicNamespaceSplit(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "monolithic_namespace_split.clj",
			RuleID:        "monolithic-namespace-split",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Use of load stitches", StartLine: 8, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Use of in-ns switches", StartLine: 14, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Use of load stitches", StartLine: 19, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Use of in-ns switches", StartLine: 20, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Use of load stitches", StartLine: 24, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 29}, {StartLine: 30},
				{StartLine: 33}, {StartLine: 34},
				{StartLine: 38}, {StartLine: 42},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
