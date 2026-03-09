package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestImplicitNamespaceDependencies(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "implicit_namespace_dependencies.clj",
			RuleID:        "implicit-namespace-dependencies",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Implicit namespace dependency: :use directive", StartLine: 2},
				{Message: "Implicit namespace dependency: :use directive", StartLine: 3},
				{Message: "Implicit namespace dependency: :refer :all", StartLine: 4},
				{Message: "Implicit namespace dependency: standalone (use", StartLine: 9},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
