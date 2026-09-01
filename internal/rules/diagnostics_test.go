package rules

import "testing"

func TestSemanticDiagnosticsCountsFindingsAndMissingEvidence(t *testing.T) {
	diagnostics := NewSemanticDiagnostics()
	diagnostics.RecordFinding(&Finding{RuleID: "sample", Contextual: true, MissingEvidence: []string{"type", "type", "contract"}})
	diagnostics.RecordFinding(&Finding{RuleID: "sample"})
	diagnostics.RecordBlocked("sample", "ownership")

	report := diagnostics.Report()
	entry := report.Rules["sample"]
	if entry.ObservedCandidates != 2 || entry.ContextualFindings != 1 || entry.ProvenFindings != 1 || entry.BlockedCandidates != 2 {
		t.Fatalf("unexpected diagnostics counters: %#v", entry)
	}
	if entry.BlockedByEvidence["type"] != 1 || entry.BlockedByEvidence["contract"] != 1 || entry.BlockedByEvidence["ownership"] != 1 {
		t.Fatalf("unexpected missing-evidence counters: %#v", entry.BlockedByEvidence)
	}
}

func TestSemanticDiagnosticsMergeIsDeterministicAndIndependent(t *testing.T) {
	left := SemanticDiagnosticsReport{}
	right := SemanticDiagnosticsReport{Rules: map[string]SemanticRuleDiagnostics{
		"b-rule": {ObservedCandidates: 2, BlockedByEvidence: map[string]int{"z": 1}},
		"a-rule": {ProvenFindings: 3},
	}}
	left.Merge(right)
	right.Rules["b-rule"].BlockedByEvidence["z"] = 99
	if left.Rules["b-rule"].BlockedByEvidence["z"] != 1 {
		t.Fatal("merge must not alias incoming evidence maps")
	}
	ids := left.SortedRuleIDs()
	if len(ids) != 2 || ids[0] != "a-rule" || ids[1] != "b-rule" {
		t.Fatalf("rule IDs are not sorted deterministically: %#v", ids)
	}
}
