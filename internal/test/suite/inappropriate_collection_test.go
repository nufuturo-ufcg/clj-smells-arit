package suite

import (
	"testing"

	"github.com/thlaurentino/arit/internal/test/framework"
)

func TestInappropriateCollection(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "inappropriate_collection.clj",
			RuleID:        "inappropriate-collection",
			ExpectedFindings: []framework.ExpectedFinding{
				{Message: "Using 'nth' for random access on a list is inefficient", StartLine: 10},
				{Message: "Using 'some' for membership check on a vector is inefficient", StartLine: 14},
				{Message: "Using 'contains?' with a non-numeric key on a Vector is inefficient", StartLine: 18},
				{Message: "Using '(first (filter ...))' is inefficient", StartLine: 26},
				{Message: "Using '(empty? (filter ...))' is inefficient", StartLine: 31},
				{Message: "Using '(count (filter ...))' processes entire collection", StartLine: 36},
				{Message: "Using '(sequence (mapcat ...))' can cause memory issues", StartLine: 45},
				{Message: "Nested 'concat' operations can be inefficient", StartLine: 50},
				{Message: "Nested 'concat' operations can be inefficient", StartLine: 50},
				{Message: "Using 'apply concat' is inefficient", StartLine: 55},
				{Message: "Using 'map' with 'identity' is unnecessary", StartLine: 64},
				{Message: "Using 'reverse' on a lazy sequence forces full realization", StartLine: 69},
				{Message: "Simple 'for' comprehension can be replaced with 'map'", StartLine: 74},
				{Message: "Using 'filter' with '(comp not predicate)' is less clear", StartLine: 83},
				{Message: "Using 'remove' with '(comp not predicate)' creates double negation", StartLine: 88},
				{Message: "Using 'into []' is less clear than using 'vec'", StartLine: 97},
				{Message: "Using 'into #{}' is less clear than using 'set'", StartLine: 102},
				{Message: "Using 'doall' with 'map' forces full realization", StartLine: 111},
				{Message: "Using '(= 0 (count coll))' is less idiomatic", StartLine: 120},
				{Message: "Using '(> (count coll) 0)' is less idiomatic", StartLine: 125},
				{Message: "Using '(not (empty? coll))' is less idiomatic", StartLine: 130},
				{Message: "Using 'merge' with many small maps can be inefficient", StartLine: 139},
				{Message: "Using 'assoc-in' with a single key is unnecessary overhead", StartLine: 148},
				{Message: "Using 'get-in' with a single key is unnecessary overhead", StartLine: 153},
				{Message: "Using 'zipmap' with 'range' is inefficient", StartLine: 162},
				{Message: "Using '(take n (repeatedly f))' may be less clear", StartLine: 171},
				{Message: "Using 'keys' on 'group-by' result just to get distinct values", StartLine: 180},
				{Message: "Using '(not (zero? (count coll)))' is less idiomatic", StartLine: 189},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}
