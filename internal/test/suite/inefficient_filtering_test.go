package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestInefficientFiltering(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "inefficient_filtering.clj",
			RuleID:        "inefficient-filtering",
			ExpectedFindings: []framework.ExpectedFinding{
				// GROUP 1: COMMON INEFFICIENT FILTERING SMELLS
				{Message: "Using `(first (filter ...))` is inefficient. Consider using `(some <predicate> <collection>)` which stops after the first match.", StartLine: 22},
				{Message: "Using `(count (filter ...))` creates an intermediate collection just to count. Consider using `reduce` with a counter or `transduce` for better performance.", StartLine: 26},
				{Message: "Using `(empty? (filter ...))` processes the entire collection. Consider using `(not (some <predicate> <collection>))` which stops early.", StartLine: 30},
				{Message: "Multiple consecutive filter operations detected in threading macro. Consider combining predicates into a single filter for better performance: `(filter #(and (pred1 %) (pred2 %)) coll)`", StartLine: 34},
				{Message: "Using `(not-empty (filter ...))` processes the entire collection. Consider using `(some <predicate> <collection>)` which returns the first truthy result or nil.", StartLine: 40},
				{Message: "Using `(count (filter ...))` creates an intermediate collection just to count. Consider using `reduce` with a counter or `transduce` for better performance.", StartLine: 44},

				// GROUP 2: INTERMEDIATE-LEVEL FILTERING SMELLS
				{Message: "Using `(map ... (filter ...))` creates an intermediate collection. Consider using `(keep ...)` which combines filtering and mapping in one pass.", StartLine: 53},
				{Message: "Filter followed by remove (or vice versa) creates two passes through the collection. Consider combining into a single filter with a compound predicate for better performance.", StartLine: 57},
				{Message: "Using `(take n (filter ...))` filters the entire collection before taking. Consider using transducers `(sequence (comp (filter ...) (take n)) coll)` for better performance.", StartLine: 63},
				{Message: "Using `(map ... (filter ...))` creates an intermediate collection. Consider using `(keep ...)` which combines filtering and mapping in one pass.", StartLine: 67},
				{Message: "Using `(first (filter ...))` is inefficient. Consider using `(some <predicate> <collection>)` which stops after the first match.", StartLine: 71},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
