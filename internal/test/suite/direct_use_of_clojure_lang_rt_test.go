package suite

import (
	"testing"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestDirectUseOfClojureLangRT(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "direct_use_of_clojure_lang_rt.clj",
			RuleID:        "direct-use-of-clojure-lang-rt",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Direct usage of clojure.lang.RT detected: 'clojure.lang.RT/count'", StartLine: 4},
				{Message: "Direct usage of clojure.lang.RT detected: 'RT/get'", StartLine: 7},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
