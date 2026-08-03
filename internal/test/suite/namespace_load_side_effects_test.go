package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestNamespaceLoadSideEffects(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "namespace_load_side_effects.clj",
			RuleID:       "namespace-load-side-effects",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Side effect: 'require' detected outside of ns macro.", StartLine: 8},  
				{Message: "Side effect: 'requiring-resolve' detected outside of ns macro.", StartLine: 11}, 
				{Message: "Side effect: 'require' detected outside of ns macro.", StartLine: 15},
				{Message: "Side effect: 'requiring-resolve' detected outside of ns macro.", StartLine: 19},
				{Message: "Side effect: 'require' detected outside of ns macro.", StartLine: 23},
				{Message: "Side effect: 'require' detected outside of ns macro.", StartLine: 27},
				{Message: "Side effect: 'require' detected outside of ns macro.", StartLine: 32},
				{Message: "Side effect: 'require' detected outside of ns macro.", StartLine: 37},
				{Message: "Side effect: 'carregar-modulo' detected outside of ns macro.", StartLine: 41}, // Não foi identificado
				{Message: "Side effect: 'require' detected outside of ns macro.", StartLine: 45},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
