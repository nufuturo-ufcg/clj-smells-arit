package suite

import (
	"testing"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestNamespaceLoadSideEffects(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "namespace_load_side_effects.clj",
			RuleID:        "namespace-load-side-effects",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Side effect: 'require' detected outside of ns macro.", StartLine: 3},
				{Message: "Side effect: 'requiring-resolve' detected outside of ns macro.", StartLine: 6},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
