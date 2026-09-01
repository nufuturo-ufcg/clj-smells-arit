package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestNonIdiomaticRecordConstruction(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "non_idiomatic_record_construction.clj",
			RuleID:        "non-idiomatic-record-construction",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Using Java interop syntax to instantiate the defrecord instead of ->User or map->User", StartLine: 10, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Using Java interop syntax to instantiate the defrecord instead of ->Account or map->Account", StartLine: 31, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Using Java interop syntax to instantiate the defrecord instead of ->Person or map->Person", StartLine: 41, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Using Java interop syntax to instantiate the defrecord instead of ->Address or map->Address", StartLine: 53, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Using Java interop syntax to instantiate the defrecord instead of ->Task or map->Task", StartLine: 60, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Using Java interop syntax to instantiate the defrecord instead of ->Product or map->Product", StartLine: 68, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Using Java interop syntax to instantiate the defrecord instead of ->Profile or map->Profile", StartLine: 75, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 17},
				{StartLine: 24},
				{StartLine: 47},
				{StartLine: 83},
				{StartLine: 87},
				{StartLine: 133},
				{StartLine: 140},
				{StartLine: 146},
				{StartLine: 150},
			},
		},
		{
			FileToAnalyze: "non_idiomatic_record_construction_arity.clj",
			RuleID:        "non-idiomatic-record-construction",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Using Java interop syntax to instantiate the defrecord instead of ->User or map->User", StartLine: 10, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 5},
				{StartLine: 6},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
