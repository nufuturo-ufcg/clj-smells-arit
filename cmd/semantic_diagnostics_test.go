package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
)

func TestWriteSemanticDiagnostics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "semantic-diagnostics.json")
	report := rules.SemanticDiagnosticsReport{Rules: map[string]rules.SemanticRuleDiagnostics{
		"sample": {ObservedCandidates: 1, ContextualFindings: 1, BlockedCandidates: 1, BlockedByEvidence: map[string]int{"contract": 1}},
	}}
	if err := writeSemanticDiagnostics(path, report); err != nil {
		t.Fatal(err)
	}
	data := readTestFile(t, path)
	var decoded rules.SemanticDiagnosticsReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Rules["sample"].BlockedByEvidence["contract"] != 1 {
		t.Fatalf("unexpected diagnostics report: %#v", decoded)
	}
}

func TestWriteSemanticDiagnosticsRejectsStdout(t *testing.T) {
	if err := writeSemanticDiagnostics("-", rules.SemanticDiagnosticsReport{}); err == nil {
		t.Fatal("expected stdout path to be rejected")
	}
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
