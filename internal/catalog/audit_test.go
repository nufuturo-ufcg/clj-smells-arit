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
