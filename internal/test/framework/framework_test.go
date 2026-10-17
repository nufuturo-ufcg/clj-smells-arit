package framework

import (
	"testing"

	rules "github.com/thlaurentino/arit/internal/rules"
)

func TestFrameworkAssertNoFindings(t *testing.T) {
	tc := ProjectTestCase{
		Files: map[string]string{
			"clean.clj": "(ns clean)\n(defn clean-func [x] (inc x))",
		},
		EntryPoint:       "clean.clj",
		RuleID:           "redundant-do-block",
		AssertNoFindings: true,
	}
	RunProjectTest(t, tc)
}

func TestFrameworkProjectTestCase(t *testing.T) {
	tc := ProjectTestCase{
		DataFiles: []string{
			"implicit_namespace_dependencies_project_provider.clj",
			"implicit_namespace_dependencies_project_missing.clj",
		},
		EntryPoint: "implicit_namespace_dependencies_project_missing.clj",
		RuleID:     "implicit-namespace-dependencies",
		ExpectedFindings: []ExpectedFinding{
			{
				StartLine:               5,
				RequireConfidence:       rules.ConfidenceContextual,
				RequireContextual:       true,
				ExpectedMissingEvidence: []string{"namespace-loading-contract", "project-dependency-index"},
			},
		},
	}
	RunProjectTest(t, tc)
}
