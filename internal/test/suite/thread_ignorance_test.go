package suite

import (
	"github.com/thlaurentino/arit/internal/test/framework"
	"testing"
)

func TestThreadIgnorance(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "thread_ignorance.clj",
			RuleID:        "thread-ignorance",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Safe ->> pipeline detected across 4 resolved calls", StartLine: 5},
				{Message: "Safe -> pipeline detected across 4 resolved calls", StartLine: 8},
				{Message: "Safe ->> pipeline detected in 3 let bindings", StartLine: 11},
				{Message: "Safe -> pipeline detected across 4 resolved calls", StartLine: 17},
			},
		},
		{
			FileToAnalyze:    "thread_ignorance_precision.clj",
			RuleID:           "thread-ignorance",
			ExpectedFindings: nil,
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 6},
				{StartLine: 10},
				{StartLine: 14},
				{StartLine: 18},
				{StartLine: 22},
				{StartLine: 29},
				{StartLine: 36},
				{StartLine: 43},
				{StartLine: 50},
				{StartLine: 57},
				{StartLine: 66},
				{StartLine: 73},
				{StartLine: 74},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
