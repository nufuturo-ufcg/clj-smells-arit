package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestDirectUseOfClojureLangRT(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "direct_use_of_clojure_lang_rt.clj",
			RuleID:        "direct-use-of-clojure-lang-rt",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Direct usage of clojure.lang.RT detected: 'clojure.lang.RT/count'", StartLine: 5, Severity: rules.SeverityHint},
				{Message: "Direct usage of clojure.lang.RT detected: 'RT/get'", StartLine: 8},
				{Message: "Direct usage of clojure.lang.RT detected: 'clojure.lang.RT/count'", StartLine: 18},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 12},
				{StartLine: 15},
				{StartLine: 21},
			},
		},
		{
			FileToAnalyze:     "direct_use_of_clojure_lang_rt_alias.clj",
			RuleID:            "direct-use-of-clojure-lang-rt",
			ExpectedFindings:  []framework.ExpectedFinding{},
			ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 5}},
		},
		{
			FileToAnalyze:     "direct_use_of_clojure_lang_rt_unresolved.clj",
			RuleID:            "direct-use-of-clojure-lang-rt",
			ExpectedFindings:  []framework.ExpectedFinding{},
			ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 4}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
