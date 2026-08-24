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
				{Message: "Namespace load side effect: 'require'", StartLine: 15},
				{Message: "Namespace load side effect: 'requiring-resolve'", StartLine: 19},
				{Message: "Namespace load side effect: 'require'", StartLine: 23},
				{Message: "Namespace load side effect: 'require'", StartLine: 27},
				{Message: "Namespace load side effect: 'require'", StartLine: 32},
				{Message: "Namespace load side effect: 'require'", StartLine: 45},
				{Message: "Namespace load side effect: 'require'", StartLine: 106},
				{Message: "Namespace load side effect: 'clojure.core/require'", StartLine: 118},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 57},
				{StartLine: 58},
				{StartLine: 62},
				{StartLine: 66},
				{StartLine: 71},
				{StartLine: 75},
				{StartLine: 79},
				{StartLine: 83},
				{StartLine: 87},
				{StartLine: 92},
				{StartLine: 96},
				{StartLine: 101},
				{StartLine: 112},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze:     "namespace_load_side_effects_fn_literal.clj",
		RuleID:            "namespace-load-side-effects",
		ExpectedFindings:  []framework.ExpectedFinding{},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 4}},
	})
}
