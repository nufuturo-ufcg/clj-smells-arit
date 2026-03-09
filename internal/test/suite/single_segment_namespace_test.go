package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestSingleSegmentNamespace(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "single_segment_namespace.clj",           
            RuleID:        "single-segment-namespace",              
            ExpectedFindings: []framework.ExpectedFinding{
                {
                    Message:   "Single-segment namespace",     
                    StartLine: 10,                     
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