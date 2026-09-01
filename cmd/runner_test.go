package cmd

import (
	"testing"

	"github.com/thlaurentino/arit/internal/config"
)

func TestFilterTestFilesExcludesTestPathsByDefault(t *testing.T) {
	oldAnalyzeTestsFlag := analyzeTestsFlag
	analyzeTestsFlag = false
	t.Cleanup(func() { analyzeTestsFlag = oldAnalyzeTestsFlag })

	files := []string{
		"src/core.clj",
		"internal/test/data/example.clj",
		"src/example_test.clj",
		"src/testing_helpers.clj",
	}

	filtered := filterTestFiles(files, &config.Config{})
	want := []string{"src/core.clj", "src/testing_helpers.clj"}

	if len(filtered) != len(want) {
		t.Fatalf("got %v, want %v", filtered, want)
	}
	for i := range want {
		if filtered[i] != want[i] {
			t.Errorf("filtered[%d] = %q, want %q", i, filtered[i], want[i])
		}
	}
}

func TestFilterTestFilesIncludesTestsWithFlag(t *testing.T) {
	oldAnalyzeTestsFlag := analyzeTestsFlag
	analyzeTestsFlag = true
	t.Cleanup(func() { analyzeTestsFlag = oldAnalyzeTestsFlag })

	files := []string{"internal/test/data/example.clj", "src/example_test.clj"}
	filtered := filterTestFiles(files, &config.Config{})

	if len(filtered) != len(files) {
		t.Fatalf("got %v, want all files %v", filtered, files)
	}
}

func TestFilterTestFilesRespectsConfig(t *testing.T) {
	oldAnalyzeTestsFlag := analyzeTestsFlag
	analyzeTestsFlag = false
	t.Cleanup(func() { analyzeTestsFlag = oldAnalyzeTestsFlag })

	files := []string{"internal/test/data/example.clj"}
	filtered := filterTestFiles(files, &config.Config{AnalyzeTests: true})

	if len(filtered) != 1 || filtered[0] != files[0] {
		t.Fatalf("got %v, want %v", filtered, files)
	}
}

func TestAutomaticWorkerCountPreservesSafeBounds(t *testing.T) {
	tests := []struct {
		name  string
		files int
		cpus  int
		want  int
	}{
		{name: "small corpus caps at eight", files: 10, cpus: 32, want: 8},
		{name: "medium corpus caps at twelve", files: 101, cpus: 32, want: 12},
		{name: "large corpus caps at sixteen", files: 501, cpus: 32, want: 16},
		{name: "zero cpus remains usable", files: 1, cpus: 0, want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := automaticWorkerCount(test.files, test.cpus); got != test.want {
				t.Fatalf("automaticWorkerCount(%d, %d) = %d, want %d", test.files, test.cpus, got, test.want)
			}
		})
	}
}
