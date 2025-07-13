package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnnecessaryInto(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "unnecessary_into.clj",
			RuleID:        "unnecessary-into",
			ExpectedFindings: []framework.ExpectedFinding{
				// TYPE TRANSFORMATION
				{Message: "Unnecessary use of 'into' for type transformation.", StartLine: 7},  //to-vector
				{Message: "Unnecessary use of 'into' for type transformation.", StartLine: 11}, //to-into
				{Message: "Unnecessary use of 'into' for type transformation.", StartLine: 27}, //flatten-lists
				{Message: "Unnecessary use of 'into' for type transformation.", StartLine: 31}, //reverse-vector
				{Message: "Unnecessary use of 'into' for type transformation.", StartLine: 35}, //get-keys
				{Message: "Unnecessary use of 'into' for type transformation.", StartLine: 47}, //map-to-pairs

				// MAP MAPPING
				{Message: "Inefficient map mapping with 'into'.", StartLine: 15}, //transform-map-values -- ERRO NA IDENTIFICAÇÃO

				// TRANSDUCER API
				{Message: "Consider using transducer API for better performance.", StartLine: 19}, //filter-evens -- ERRO NA IDENTIFICAÇÃO
				{Message: "Consider using transducer API for better performance.", StartLine: 23}, //increment-all -- ERRO NA IDENTIFICAÇÃO
				{Message: "Consider using transducer API for better performance.", StartLine: 39}, //map-append
				{Message: "Consider using transducer API for better performance.", StartLine: 43}, //filter-positive
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
