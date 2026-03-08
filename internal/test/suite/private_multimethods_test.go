package suite
import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)
func TestPrivateMultimethods(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "private_multimethods.clj",
            RuleID:        "private-multimethods",
            ExpectedFindings: []framework.ExpectedFinding{
                {Message: "Private multimethod", StartLine: 6},
                {Message: "Private multimethod", StartLine: 15},
                {Message: "Private multimethod", StartLine: 37},
                {Message: "Private multimethod", StartLine: 46},
                {Message: "Private multimethod", StartLine: 59},
                //{Message: "Private multimethod", StartLine: 67},
                {Message: "Private multimethod", StartLine: 73},
            },
        },
    }
    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.DebugRuleTest(t, tc)
        })
    }
}
