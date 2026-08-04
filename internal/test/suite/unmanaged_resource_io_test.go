package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnmanagedResourceIo(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "unmanaged_resource_io.clj",
			RuleID:       "unmanaged-resource-io",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "I/O resource used without with-open: use with-open to ensure the resource is closed.", StartLine: 7},  
				{Message: "I/O resource used without with-open: use with-open to ensure the resource is closed.", StartLine: 13}, 
				{Message: "I/O resource used without with-open: use with-open to ensure the resource is closed.", StartLine: 18},
				{Message: "I/O resource used without with-open: use with-open to ensure the resource is closed.", StartLine: 23},
				{Message: "I/O resource used without with-open: use with-open to ensure the resource is closed.", StartLine: 29},
				{Message: "I/O resource used without with-open: use with-open to ensure the resource is closed.", StartLine: 34},
				{Message: "I/O resource used without with-open: use with-open to ensure the resource is closed.", StartLine: 40}, 
				{Message: "I/O resource used without with-open: use with-open to ensure the resource is closed.", StartLine: 45},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}