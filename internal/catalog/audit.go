// Package catalog provides regression auditing for the synthetic Clojure smell catalog.
package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/rules"
	_ "github.com/thlaurentino/arit/internal/rules/clojurespecific"
	_ "github.com/thlaurentino/arit/internal/rules/functional"
	"github.com/thlaurentino/arit/internal/rules/semantics"
	"gopkg.in/yaml.v3"
)

type Expectation struct {
	Status           string   `yaml:"status"`
	NoFindingReason  string   `yaml:"no-finding-reason"`
	ExpectedFindings []string `yaml:"expected-findings"`
	Classification   string   `yaml:"classification"`
	NextAction       string   `yaml:"next-action"`
}

type DirectoryExpectation struct {
	Path        string `yaml:"path"`
	Expectation `yaml:",inline"`
}

type FileExpectation struct {
	Path        string `yaml:"path"`
	Expectation `yaml:",inline"`
}

type Manifest struct {
	Version     int                    `yaml:"version"`
	Dataset     string                 `yaml:"dataset"`
	Defaults    Expectation            `yaml:"defaults"`
	Directories []DirectoryExpectation `yaml:"directories"`
	Files       []FileExpectation      `yaml:"files"`
}

type Options struct {
	ExperimentalCrossNamespace bool
}

type FileReport struct {
	Path        string      `json:"path"`
	Expectation Expectation `json:"expectation"`
	Findings    int         `json:"findings"`
	Proven      int         `json:"proven"`
	Contextual  int         `json:"contextual"`
	NoFinding   bool        `json:"no_finding"`
}

type Report struct {
	Root                    string       `json:"root"`
	ManifestPath            string       `json:"manifest_path"`
	Files                   []FileReport `json:"files"`
	Findings                int          `json:"findings"`
	Proven                  int          `json:"proven"`
	Contextual              int          `json:"contextual"`
	NoFindingFiles          []FileReport `json:"no_finding_files"`
	ContextualWithoutReason []string     `json:"contextual_without_reason"`
	Violations              []string     `json:"violations"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	if manifest.Defaults.Status == "" {
		manifest.Defaults.Status = "may-find"
	}
	return manifest, nil
}

func (manifest Manifest) ExpectationFor(path string) Expectation {
	normalized := filepath.ToSlash(path)
	for _, entry := range manifest.Files {
		if filepath.ToSlash(entry.Path) == normalized {
			return entry.Expectation
		}
	}
	bestLength := -1
	selected := manifest.Defaults
	for _, entry := range manifest.Directories {
		directory := strings.TrimSuffix(filepath.ToSlash(entry.Path), "/")
		if normalized == directory || strings.HasPrefix(normalized, directory+"/") {
			if len(directory) > bestLength {
				bestLength = len(directory)
				selected = entry.Expectation
			}
		}
	}
	if selected.Status == "" {
		selected.Status = "may-find"
	}
	return selected
}

func Audit(root, manifestPath string, options Options) (Report, error) {
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return Report{}, err
	}
	files, err := clojureFiles(root)
	if err != nil {
		return Report{}, err
	}
	if len(files) == 0 {
		return Report{}, fmt.Errorf("no Clojure files found below %s", root)
	}

	report := Report{Root: root, ManifestPath: manifestPath}
	cfg := &config.Config{
		EnabledRules:  make(map[string]bool),
		EnabledGroups: make(map[string]bool),
		RuleConfig:    make(map[string]config.RuleSettings),
	}
	projectIndex := (*semantics.ProjectIndex)(nil)
	if options.ExperimentalCrossNamespace {
		projectIndex = semantics.NewProjectIndex()
		for _, path := range files {
			if err := projectIndex.IndexFile(path); err != nil {
				return Report{}, err
			}
		}
		projectIndex.BuildMacroCallSites()
		projectIndex.MarkMacroCallSitesComplete()
	}

	previousCrossNamespace := analyzer.EnableExperimentalCrossNamespace
	analyzer.EnableExperimentalCrossNamespace = options.ExperimentalCrossNamespace
	defer func() { analyzer.EnableExperimentalCrossNamespace = previousCrossNamespace }()
	instance := analyzer.NewAnalyzer(cfg)
	for _, path := range files {
		result, analyzeErr := instance.AnalyzeFileWithProjectIndex(path, projectIndex)
		if analyzeErr != nil {
			return Report{}, fmt.Errorf("analyze %s: %w", path, analyzeErr)
		}
		expectation := manifest.ExpectationFor(path)
		fileReport := FileReport{Path: path, Expectation: expectation, Findings: len(result.Findings)}
		for _, finding := range result.Findings {
			report.Findings++
			if finding.Confidence == rules.ConfidenceContextual {
				report.Contextual++
				fileReport.Contextual++
				if len(finding.MissingEvidence) == 0 && finding.ContextualReason == "" {
					report.ContextualWithoutReason = append(report.ContextualWithoutReason, fmt.Sprintf("%s:%s", path, finding.RuleID))
				}
			} else {
				report.Proven++
				fileReport.Proven++
			}
		}
		fileReport.NoFinding = len(result.Findings) == 0
		if fileReport.NoFinding {
			report.NoFindingFiles = append(report.NoFindingFiles, fileReport)
		}
		report.Files = append(report.Files, fileReport)
		validateExpectation(&report, fileReport)
	}
	sort.Slice(report.Files, func(i, j int) bool { return report.Files[i].Path < report.Files[j].Path })
	sort.Slice(report.NoFindingFiles, func(i, j int) bool { return report.NoFindingFiles[i].Path < report.NoFindingFiles[j].Path })
	return report, nil
}

func validateExpectation(report *Report, file FileReport) {
	switch file.Expectation.Status {
	case "must-find":
		if file.Findings == 0 {
			report.Violations = append(report.Violations, fmt.Sprintf("%s: expected at least one finding (%s)", file.Path, file.Expectation.NoFindingReason))
		}
	case "must-not-find":
		if file.Findings != 0 {
			report.Violations = append(report.Violations, fmt.Sprintf("%s: expected no findings", file.Path))
		}
	case "may-find", "":
		// This status is intentionally permissive for catalog entries awaiting a precise contract.
	default:
		report.Violations = append(report.Violations, fmt.Sprintf("%s: unsupported manifest status %q", file.Path, file.Expectation.Status))
	}
}

func clojureFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".clj" || ext == ".cljs" || ext == ".cljc" {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk catalog: %w", err)
	}
	sort.Strings(files)
	return files, nil
}
