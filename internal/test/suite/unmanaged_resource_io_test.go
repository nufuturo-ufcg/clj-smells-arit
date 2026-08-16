package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnmanagedResourceIo(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "unmanaged_resource_io.clj",
			RuleID:        "unmanaged-resource-io",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Resource created by `clojure.java.io/reader` is bound to `r`", StartLine: 7},
				{Message: "Resource created by `clojure.java.io/output-stream` is bound to `out`", StartLine: 13},
				{Message: "Resource created by `java.io.FileReader.` is bound to `f-reader`", StartLine: 18},
				{Message: "Resource created by `java.net.ServerSocket.` is bound to `srv`", StartLine: 23},
				{Message: "Resource created by `clojure.java.io/input-stream` is bound to `stream`", StartLine: 29},
				{Message: "Resource created by `clojure.java.io/reader` is bound to `f`", StartLine: 34},
				{Message: "Resource created by `java.util.zip.GZIPInputStream.` is bound to `zip`", StartLine: 40},
				{Message: "Resource created by `java.net.Socket.` is bound to `canal`", StartLine: 45},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 53},
				{StartLine: 58},
				{StartLine: 59},
				{StartLine: 64},
				{StartLine: 69},
				{StartLine: 81},
				{StartLine: 85},
			},
		},
		{
			FileToAnalyze: "unmanaged_resource_io_precision.clj",
			RuleID:        "unmanaged-resource-io",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Resource created by `clojure.java.io/reader` is bound to `r`", StartLine: 10},
				{Message: "Resource created by `java.io.FileInputStream.` is bound to `in`", StartLine: 15},
				{Message: "Resource created by `java.util.zip.GZIPInputStream.` is bound to `gzip`", StartLine: 20},
				{Message: "Resource created by `java.net.ServerSocket.` is bound to `server`", StartLine: 88},
			},
			ForbiddenFindings: []framework.ExpectedFinding{
				{StartLine: 26},
				{StartLine: 30},
				{StartLine: 34},
				{StartLine: 38},
				{StartLine: 42},
				{StartLine: 47},
				{StartLine: 52},
				{StartLine: 57},
				{StartLine: 62},
				{StartLine: 67},
				{StartLine: 72},
				{StartLine: 83},
				{StartLine: 93},
				{StartLine: 94},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
