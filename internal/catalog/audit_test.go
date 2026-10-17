package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpectationForUsesExactFileThenLongestDirectoryPrefix(t *testing.T) {
	manifest := Manifest{
		Defaults: Expectation{Status: "may-find"},
		Directories: []DirectoryExpectation{
			{Path: "catalog", Expectation: Expectation{Status: "must-find"}},
			{Path: "catalog/negative", Expectation: Expectation{Status: "must-not-find"}},
		},
		Files: []FileExpectation{
			{Path: "catalog/negative/known.clj", Expectation: Expectation{Status: "may-find"}},
		},
	}

	if got := manifest.ExpectationFor("catalog/example.clj").Status; got != "must-find" {
		t.Fatalf("directory expectation = %q, want must-find", got)
	}
	if got := manifest.ExpectationFor("catalog/negative/example.clj").Status; got != "must-not-find" {
		t.Fatalf("longest directory expectation = %q, want must-not-find", got)
	}
	if got := manifest.ExpectationFor("catalog/negative/known.clj").Status; got != "may-find" {
		t.Fatalf("exact file expectation = %q, want may-find", got)
	}
}

func TestAuditReportsNoFindingFilesExplicitly(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "example.clj")
	if err := os.WriteFile(file, []byte("(ns fixture.core)\n(def value 1)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "manifest.yaml")
	manifest := []byte("version: 1\ndefaults:\n  status: may-find\n  no-finding-reason: fixture is intentionally negative\n")
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := Audit(root, manifestPath, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.NoFindingFiles) != 1 {
		t.Fatalf("no-finding files = %d, want 1", len(report.NoFindingFiles))
	}
	if report.NoFindingFiles[0].Expectation.NoFindingReason == "" {
		t.Fatal("no-finding report did not retain its explanation")
	}
}

func TestAuditValidatesExpectedConfidence(t *testing.T) {
	root := t.TempDir()
	// Create a file with a redundant-do-block finding (proven)
	file := filepath.Join(root, "proven_example.clj")
	content := []byte("(ns fixture.core)\n(defn foo [] (let [x 1] (do 1 2)))\n")
	if err := os.WriteFile(file, content, 0o600); err != nil {
		t.Fatal(err)
	}

	// Manifest expecting contextual confidence on a proven finding
	manifestPath := filepath.Join(root, "manifest.yaml")
	manifest := []byte(`version: 1
defaults:
  status: may-find
files:
  - path: proven_example.clj
    status: must-find
    expected-confidence: contextual
`)
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := Audit(root, manifestPath, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Violations) == 0 {
		t.Fatal("expected violation for confidence mismatch, got none")
	}
}

func TestAuditValidatesMissingEvidence(t *testing.T) {
	root := t.TempDir()
	// Create a file with single-segment namespace (contextual, with evidence "namespace-scope-contract" and "deployment-context")
	file := filepath.Join(root, "single_segment.clj")
	content := []byte("(ns single-segment)\n(def x 1)\n")
	if err := os.WriteFile(file, content, 0o600); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(root, "manifest.yaml")
	manifest := []byte(`version: 1
defaults:
  status: may-find
files:
  - path: single_segment.clj
    status: must-find
    expected-confidence: contextual
    missing-evidence:
      - non-existent-evidence-token
`)
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := Audit(root, manifestPath, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Violations) == 0 {
		t.Fatal("expected violation for missing evidence token mismatch, got none")
	}
}

func TestAuditDistinguishesGateAndExploratoryRoles(t *testing.T) {
	root := t.TempDir()
	fileNeg := filepath.Join(root, "negative.clj")
	if err := os.WriteFile(fileNeg, []byte("(ns fixture.core)\n(def x 1)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	filePos := filepath.Join(root, "positive.clj")
	if err := os.WriteFile(filePos, []byte("(ns fixture.core)\n(defn foo [] (let [x 1] (do 1 2)))\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileExp := filepath.Join(root, "exploratory.clj")
	if err := os.WriteFile(fileExp, []byte("(ns fixture.core)\n(def y 2)\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(root, "manifest.yaml")
	manifest := []byte(`version: 1
defaults:
  status: may-find
files:
  - path: negative.clj
    status: must-not-find
  - path: positive.clj
    status: must-find
    expected-confidence: proven
  - path: exploratory.clj
    status: may-find
`)
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := Audit(root, manifestPath, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Violations) > 0 {
		t.Fatalf("unexpected violations: %v", report.Violations)
	}
	if report.GateFiles != 2 {
		t.Fatalf("gate files = %d, want 2", report.GateFiles)
	}
	if report.ExploratoryFiles != 1 {
		t.Fatalf("exploratory files = %d, want 1", report.ExploratoryFiles)
	}
	if report.TruePositives != 1 {
		t.Fatalf("true positives = %d, want 1", report.TruePositives)
	}
	if report.TrueNegatives != 1 {
		t.Fatalf("true negatives = %d, want 1", report.TrueNegatives)
	}
	if report.FalsePositives != 0 || report.FalseNegatives != 0 {
		t.Fatalf("false alarms present: fp=%d fn=%d", report.FalsePositives, report.FalseNegatives)
	}
}

// A rule-scoped expectation ignores findings of other rules and checks the
// exact count of findings of its own rule.
func TestAuditScopesExpectationToRuleAndCount(t *testing.T) {
	root := t.TempDir()
	// single-segment-namespace fires on `(ns fixture)`; redundant-do-block fires twice.
	source := "(ns fixture)\n(defn foo [] (let [x 1] (do 1 2)) (let [y 2] (do 3 4)))\n"
	for _, name := range []string{"scoped_negative.clj", "scoped_count.clj", "scoped_ok.clj"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifest := []byte(`version: 1
defaults:
  status: may-find
files:
  - path: ` + filepath.Join(root, "scoped_negative.clj") + `
    status: must-not-find
    role: gate
    rule: marker-protocol
  - path: ` + filepath.Join(root, "scoped_count.clj") + `
    status: must-find
    role: gate
    rule: redundant-do-block
    expected-count: 1
  - path: ` + filepath.Join(root, "scoped_ok.clj") + `
    status: must-find
    role: gate
    rule: redundant-do-block
    expected-count: 2
    expected-confidence: proven
`)
	manifestPath := filepath.Join(root, "manifest.yaml")
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Audit(root, manifestPath, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.TrueNegatives != 1 {
		t.Errorf("findings of other rules must not break a rule-scoped negative: TN=%d violations=%v", report.TrueNegatives, report.Violations)
	}
	if report.FalsePositives != 1 || len(report.Violations) != 1 {
		t.Fatalf("expected exactly one over-reporting violation, got FP=%d violations=%v", report.FalsePositives, report.Violations)
	}
}
