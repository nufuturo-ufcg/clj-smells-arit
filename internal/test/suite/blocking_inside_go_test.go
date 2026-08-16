package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestBlockingInsideGo(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "blocking_inside_go.clj",
			RuleID:        "blocking-inside-go",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 9},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 15},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 20},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 26},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 32},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 39},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 46},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 56},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 65},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 71},
				{Message: "Blocking function detected within the GO block a/go.", StartLine: 78},
				{Message: "Blocking function detected within the GO block legacy-async/go.", StartLine: 151},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 100},
				{StartLine: 106},
				{StartLine: 112},
				{StartLine: 118},
				{StartLine: 123},
				{StartLine: 129},
				{StartLine: 146},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
