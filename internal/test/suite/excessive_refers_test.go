package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestExcessiveRefers(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "excessive_refers.clj",
			RuleID:       "excessive-refers",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "The excessive number of explicit references in the com.my-app.bloated-utils-list namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 8},  
				{Message: "The excessive number of explicit references in the com.my-app.successive-pollution namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 13}, 
				{Message: "The excessive number of explicit references in the com.my-app.legacy-use-bloated namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 23},
				{Message: "The excessive number of explicit references in the com.my-app.edge-case-stacking namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 28},
				{Message: "The excessive number of explicit references in the com.my-app.config-overload namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 35},
				{Message: "The excessive number of explicit references in the com.my-app.service.billing namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 40},
				{Message: "The excessive number of explicit references in the com.my-app.handler.user namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 45},
				{Message: "The excessive number of explicit references in the com.my-app.checkout.flow namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 50},
				{Message: "The excessive number of explicit references in the com.my-app.services.mocked namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 57},
				{Message: "The excessive number of explicit references in the com.my-app.view.formatter namespace increases the risk of conflicts with other libraries or with future code.", StartLine: 62},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}