package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestTrivialLambda(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "trivial_lambda.clj",
            RuleID:        "trivial-lambda",
            ExpectedFindings: []framework.ExpectedFinding{
                {Message: "Trivial lambda/fn. Consider us", StartLine: 7},
                {Message: "Trivial lambda/fn. Consider us", StartLine: 11},
                {Message: "Trivial lambda/fn. Consider us", StartLine: 15},
                {Message: "Trivial lambda/fn. Consider us", StartLine: 20},
                {Message: "Trivial lambda/fn. Consider us", StartLine: 24},
                {Message: "Trivial lambda/fn. Consider us", StartLine: 28},
                {Message: "Trivial lambda/fn. Consider us", StartLine: 32},
                {Message: "Trivial lambda/fn. Consider us", StartLine: 37},
                {Message: "Trivial lambda/fn. Consider us", StartLine: 38},
                {Message: "Trivial lambda/fn. Consider us", StartLine: 43},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}

