package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnmanagedResourceIo(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "unmanaged_resource_io.clj",
			RuleID:        "unmanaged-resource-io",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Resource created by `clojure.java.io/reader` is bound to `r`", StartLine: 7, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Resource created by `clojure.java.io/output-stream` is bound to `out`", StartLine: 13, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Resource created by `java.io.FileReader.` is bound to `f-reader`", StartLine: 18, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Resource created by `java.net.ServerSocket.` is bound to `srv`", StartLine: 23, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Resource created by `clojure.java.io/input-stream` is bound to `stream`", StartLine: 29, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Resource created by `clojure.java.io/reader` is bound to `f`", StartLine: 34, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Resource created by `java.util.zip.GZIPInputStream.` is bound to `zip`", StartLine: 40, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Resource created by `java.net.Socket.` is bound to `canal`", StartLine: 45, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
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
				{Message: "Resource created by `clojure.java.io/reader` is bound to `r`", StartLine: 10, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Resource created by `java.io.FileInputStream.` is bound to `in`", StartLine: 15, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Resource created by `java.util.zip.GZIPInputStream.` is bound to `gzip`", StartLine: 20, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "Resource created by `java.net.ServerSocket.` is bound to `server`", StartLine: 93, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Resource created by `java.net.ServerSocket.` is bound to `server-socket`", StartLine: 137, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Resource created by `clojure.java.io/reader` is bound to `r`", StartLine: 169, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
				{Message: "Resource created by `clojure.java.io/reader` is bound to `r`", StartLine: 173, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
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
				{StartLine: 65},
				{StartLine: 94},
				{StartLine: 107},
				{StartLine: 66},
				{StartLine: 117},
				{StartLine: 126},
				{StartLine: 131},
				{StartLine: 143},
				{StartLine: 147},
				{StartLine: 152},
				{StartLine: 158},
				{StartLine: 163},
			},
		},
		{
			// Adapters over *in*/*out*/*err* must never be closed by their user.
			FileToAnalyze: "unmanaged_resource_io_standard_streams.clj",
			RuleID:        "unmanaged-resource-io",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "bound to `w`", StartLine: 13, RequireConfidence: rules.ConfidenceContextual, RequireContextual: true},
				{Message: "bound to `w`", StartLine: 16, RequireConfidence: rules.ConfidenceProven, RequireNonContextual: true},
			},
			ForbiddenFindings: []framework.ExpectedFinding{{StartLine: 6}, {StartLine: 10}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}

	t.Run("adversarial: proxy wrapping stream transfers ownership", func(t *testing.T) {
		framework.RunProjectTest(t, framework.ProjectTestCase{
			RuleID: "unmanaged-resource-io",
			Files: map[string]string{
				"storage.clj": `(ns datascript.storage
  (:import [java.io File FileOutputStream BufferedOutputStream OutputStream]))

(defn- output-stream
  ^OutputStream [^File file]
  (let [os (FileOutputStream. file)]
    (proxy [BufferedOutputStream] [os]
      (flush [])
      (close []
        (let [this ^BufferedOutputStream this]
          (proxy-super flush)
          (proxy-super close))))))`,
			},
			EntryPoint:       "storage.clj",
			AssertNoFindings: true,
		})
	})
}
