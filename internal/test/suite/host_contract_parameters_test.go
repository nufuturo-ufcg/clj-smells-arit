package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

// A method implementation owns no contract of its own: Collection.toArray must
// fill the array it receives, and a PreparedStatement returns the connection
// it was created from, which its caller owns.
func TestHostContractParameters(t *testing.T) {
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "host_contract_parameters.clj",
		RuleID:        "immutability-violation",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "type-hinted Java array", StartLine: 13, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 8}},
	})
	framework.RunRuleTest(t, framework.RuleTestCase{
		FileToAnalyze: "host_contract_parameters.clj",
		RuleID:        "unmanaged-resource-io",
		ExpectedFindings: []framework.ExpectedFinding{
			{Message: "getConnection", StartLine: 22},
		},
		ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 18}},
	})
}
