package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestMultipleEvaluationInMacros(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "multiple_evaluation_in_macros.clj",
			RuleID:       "multiple-evaluation-in-macros",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "The macro double-eval-basic presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 6},  
				{Message: "The macro square-unhygienic presents multiple calls to the input arguments x without defining temporary local variables.", StartLine: 12}, 
				{Message: "The macro log-and-run presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 16},
				{Message: "The macro bad-bindings presents multiple calls to the input arguments val-expr without defining temporary local variables.", StartLine: 22},
				{Message: "The macro repeat-eval-loop presents multiple calls to the input arguments coll-expr without defining temporary local variables.", StartLine: 28},
				{Message: "The macro try-re-eval presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 34},
				{Message: "The macro pair-value presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 41},
				{Message: "The macro unpack-twice presents multiple calls to the input arguments items-expr without defining temporary local variables.", StartLine: 46},
				{Message: "The macro false-safe-gensym presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 50},
				{Message: "The macro assert-verbose presents multiple calls to the input arguments expr without defining temporary local variables.", StartLine: 56},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}