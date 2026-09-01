package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestBlockingInsideGo(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "blocking_inside_go.clj",
			RuleID:        "blocking-inside-go",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 9, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 15, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 20, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 26, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 32, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 39, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 46, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 51, Severity: rules.SeverityHint, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 51, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 71, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 78, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Blocking function detected within the GO block legacy-async/go.", StartLine: 151, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 100},
				{StartLine: 106},
				{StartLine: 112},
				{StartLine: 118},
				{StartLine: 123},
				{StartLine: 129},
				{StartLine: 146},
				{StartLine: 155},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "blocking_inside_go_precision.clj",
		RuleID:            "blocking-inside-go",
		ExpectedFindings:  []framework.ExpectedFinding{{Message: "Blocking function detected", StartLine: 13, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true}},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 6}, {StartLine: 9}},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "blocking_inside_go_invalid.clj",
		RuleID:            "blocking-inside-go",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 5}, {StartLine: 6}, {StartLine: 7}},
	})
}
