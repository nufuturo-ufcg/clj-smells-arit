package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestNonIdiomaticRecordConstruction(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "non_idiomatic_record_construction.clj",
			RuleID:       "non-idiomatic-record-construction",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->User", StartLine: 10},  
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->Order", StartLine: 17}, 
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->OrderMap", StartLine: 24},
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->Account", StartLine: 31},
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->Person", StartLine: 41},
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->Customer", StartLine: 47},
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->Address", StartLine: 53},
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->Task", StartLine: 60},
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->Product", StartLine: 68},
				{Message: "Using a positional constructor to instantiate the defrecord instead of map->Profile", StartLine: 75},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
