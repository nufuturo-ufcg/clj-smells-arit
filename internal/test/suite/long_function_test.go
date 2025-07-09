package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestLongFunction(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "long_function.clj",           
            RuleID:        "long-function",              
            ExpectedFindings: []framework.ExpectedFinding{
                {
                    Message:   "is too long:",     
                    StartLine: 4,                     
                },
				{
					Message:   "is too long:",     
                    StartLine: 66,                     
				},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}