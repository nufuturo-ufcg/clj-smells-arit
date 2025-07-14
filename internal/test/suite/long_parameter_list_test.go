package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestLongParameterList(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "long_parameter_list.clj",           
            RuleID:        "long-parameter-list",              
            ExpectedFindings: []framework.ExpectedFinding{
                {
                    Message:   "has too many parameters:",     
                    StartLine: 7,                     
                },
				{
					Message:   "has too many parameters:",     
                    StartLine: 21,                     
				},
				{
					Message:   "has too many parameters:",     
                    StartLine: 35,                     
				},
                {
					Message:   "has too many parameters:",     
                    StartLine: 46,                     
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