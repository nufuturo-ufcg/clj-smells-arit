package main

import (
	"strings"
	"testing"
)

func TestCompareReportsIgnoresFormattingButPreservesOrder(t *testing.T) {
	previous := `[
  {"rule_id":"a","location":{"start_line":1}},
  {"rule_id":"b","location":{"start_line":2}}
]`
	current := `[{"location":{"start_line":1},"rule_id":"a"},{"rule_id":"b","location":{"start_line":2}}]`

	result, err := compareReports(strings.NewReader(previous), strings.NewReader(current))
	if err != nil {
		t.Fatal(err)
	}
	if result.Compared != 2 || result.Added != 0 || result.Removed != 0 || result.Changed != 0 || !result.Ordered {
		t.Fatalf("unexpected comparison result: %#v", result)
	}
}

func TestCompareReportsDetectsContentAndCountChanges(t *testing.T) {
	previous := `[{"rule_id":"a"},{"rule_id":"b"}]`
	current := `[{"rule_id":"a"},{"rule_id":"c"},{"rule_id":"d"},{"rule_id":"e"}]`

	result, err := compareReports(strings.NewReader(previous), strings.NewReader(current))
	if err != nil {
		t.Fatal(err)
	}
	if result.Compared != 2 || result.Changed != 1 || result.Added != 2 || result.Removed != 0 || result.Ordered {
		t.Fatalf("unexpected comparison result: %#v", result)
	}
}

func TestCompareReportsDetectsRemovedItems(t *testing.T) {
	previous := `[{"rule_id":"a"},{"rule_id":"b"},{"rule_id":"c"}]`
	current := `[{"rule_id":"a"}]`

	result, err := compareReports(strings.NewReader(previous), strings.NewReader(current))
	if err != nil {
		t.Fatal(err)
	}
	if result.Compared != 1 || result.Changed != 0 || result.Added != 0 || result.Removed != 2 || result.Ordered {
		t.Fatalf("unexpected comparison result: %#v", result)
	}
}

func TestCompareReportsRejectsNonArray(t *testing.T) {
	if _, err := compareReports(strings.NewReader(`{"findings":[]}`), strings.NewReader(`[]`)); err == nil {
		t.Fatal("expected non-array report to be rejected")
	}
}
