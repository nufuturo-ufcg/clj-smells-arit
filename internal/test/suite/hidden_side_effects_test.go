package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestHiddenSideEffects(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "hidden_side_effects.clj",
            RuleID:        "hidden-side-effects",  
            ExpectedFindings: []framework.ExpectedFinding{
				//MAP pattern
                {Message:   "Function 'greet-user' performs side effects", StartLine: 4}, // greet-user

				//REDUCE pattern
				{Message: "Function 'accumulate' performs side effects", StartLine: 17}, // accumulate
				
				//FILTER pattern
				{Message: "Function 'side-effect-check' appears to be pure", StartLine: 26}, // side-effect-check
				{Message: "Function 'check-even' appears to be pure", StartLine: 39}, // check-even
				
				//LAZY-SEQ
				{Message: "Function 'lazy-numbers' performs", StartLine: 34}, // lazy-numbers
				{Message: "Function 'lazy-show-nums' performs", StartLine: 69}, // lazy-show-nums
				
				//FOR pattern
				{Message: "Function 'show-and-scale' performs", StartLine: 48}, // show-and-scale
				
				//COMP pattern		
				{Message: "Function 'track-and-inc' performs", StartLine: 59}, // track-and-inc
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
				framework.RunRuleTest(t, tc)
        })
    }
}
