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
	Status             string   `yaml:"status"`
	Role               string   `yaml:"role,omitempty"`
	ExpectedConfidence string   `yaml:"expected-confidence,omitempty"`
	MissingEvidence    []string `yaml:"missing-evidence,omitempty"`
	NoFindingReason    string   `yaml:"no-finding-reason"`
	ExpectedFindings   []string `yaml:"expected-findings"`
	Classification     string   `yaml:"classification"`
	NextAction         string   `yaml:"next-action"`
	// Rule scopes the expectation to one rule: only its findings count for the
	// status, confidence, evidence, and count checks. Findings of other rules
	// in the file are ignored by the gate.
	Rule string `yaml:"rule,omitempty"`
	// ExpectedCount, when set, is the exact number of in-scope findings; more
	// is reported as over-reporting (false positives), fewer as a false negative.
	ExpectedCount *int `yaml:"expected-count,omitempty"`
}

func (e Expectation) EffectiveRole(path string) string {
	if e.Role != "" {
		return e.Role
	}
	norm := filepath.ToSlash(path)
	if strings.Contains(norm, "/real_samples/") || strings.HasSuffix(norm, "/real_samples") {
		return "exploratory"
	}
	switch e.Status {
	case "must-find", "must-not-find":
		return "gate"
	case "may-find":
		return "exploratory"
	default:
		return "exploratory"
	}
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
	Path            string      `json:"path"`
	Expectation     Expectation `json:"expectation"`
	Role            string      `json:"role"`
	Findings        int         `json:"findings"`
	Proven          int         `json:"proven"`
	Contextual      int         `json:"contextual"`
	NoFinding       bool        `json:"no_finding"`
	MissingEvidence []string    `json:"missing_evidence,omitempty"`
	// OtherRuleFindings counts findings outside a rule-scoped expectation.
	OtherRuleFindings int `json:"other_rule_findings,omitempty"`
}

type Report struct {
	Root                    string       `json:"root"`
	ManifestPath            string       `json:"manifest_path"`
	Files                   []FileReport `json:"files"`
	Findings                int          `json:"findings"`
	Proven                  int          `json:"proven"`
	Contextual              int          `json:"contextual"`
	GateFiles               int          `json:"gate_files"`
	ExploratoryFiles        int          `json:"exploratory_files"`
	TruePositives           int          `json:"true_positives"`
	TrueNegatives           int          `json:"true_negatives"`
	FalsePositives          int          `json:"false_positives"`
	FalseNegatives          int          `json:"false_negatives"`
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
		entryNorm := filepath.ToSlash(entry.Path)
		if entryNorm == normalized || strings.HasSuffix(normalized, "/"+entryNorm) || strings.TrimPrefix(normalized, "./") == strings.TrimPrefix(entryNorm, "./") {
			return entry.Expectation
		}
	}
	bestLength := -1
	selected := manifest.Defaults
	for _, entry := range manifest.Directories {
		directory := strings.TrimSuffix(filepath.ToSlash(entry.Path), "/")
		if normalized == directory || strings.HasPrefix(normalized, directory+"/") || strings.Contains(normalized, "/"+directory+"/") || strings.HasSuffix(normalized, "/"+directory) {
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
		fileReport := FileReport{
			Path:        path,
			Expectation: expectation,
			Role:        expectation.EffectiveRole(path),
		}
		evidenceSet := make(map[string]bool)
		for _, finding := range result.Findings {
			report.Findings++
			inScope := expectation.Rule == "" || finding.RuleID == expectation.Rule
			if inScope {
				fileReport.Findings++
				for _, ev := range finding.MissingEvidence {
					evidenceSet[ev] = true
				}
			} else {
				fileReport.OtherRuleFindings++
			}
			if finding.Confidence == rules.ConfidenceContextual {
				report.Contextual++
				if inScope {
					fileReport.Contextual++
				}
				if len(finding.MissingEvidence) == 0 && finding.ContextualReason == "" {
					report.ContextualWithoutReason = append(report.ContextualWithoutReason, fmt.Sprintf("%s:%s", path, finding.RuleID))
				}
			} else {
				report.Proven++
				if inScope {
					fileReport.Proven++
				}
			}
		}
		for ev := range evidenceSet {
			fileReport.MissingEvidence = append(fileReport.MissingEvidence, ev)
		}
		sort.Strings(fileReport.MissingEvidence)
		fileReport.NoFinding = fileReport.Findings == 0
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
	// 1. Role accounting and gate evaluation
	if file.Role == "gate" {
		report.GateFiles++
		switch file.Expectation.Status {
		case "must-not-find":
			if file.Findings == 0 {
				report.TrueNegatives++
			} else {
				report.FalsePositives++
				report.Violations = append(report.Violations, fmt.Sprintf("%s: expected no findings (true negative), got %d findings", file.Path, file.Findings))
			}
		case "must-find":
			if file.Findings > 0 {
				report.TruePositives++
				if count := file.Expectation.ExpectedCount; count != nil && file.Findings > *count {
					report.FalsePositives++
					report.Violations = append(report.Violations, fmt.Sprintf("%s: expected %d finding(s) of the scoped rule, got %d (over-reporting)", file.Path, *count, file.Findings))
				} else if count != nil && file.Findings < *count {
					report.FalseNegatives++
					report.Violations = append(report.Violations, fmt.Sprintf("%s: expected %d finding(s) of the scoped rule, got %d", file.Path, *count, file.Findings))
				}
			} else {
				report.FalseNegatives++
				reason := file.Expectation.NoFindingReason
				if reason == "" {
					reason = "unexplained omission"
				}
				report.Violations = append(report.Violations, fmt.Sprintf("%s: expected at least one finding (true positive) but found none (%s)", file.Path, reason))
			}
		}
	} else {
		report.ExploratoryFiles++
	}

	// 2. Status validation
	switch file.Expectation.Status {
	case "must-find", "must-not-find", "may-find", "":
		// Handled or permissive
	default:
		report.Violations = append(report.Violations, fmt.Sprintf("%s: unsupported manifest status %q", file.Path, file.Expectation.Status))
	}

	// 3. ExpectedConfidence validation
	if file.Expectation.ExpectedConfidence != "" && file.Findings > 0 {
		switch file.Expectation.ExpectedConfidence {
		case "proven":
			if file.Proven == 0 {
				report.Violations = append(report.Violations, fmt.Sprintf("%s: expected proven finding, but got %d contextual finding(s)", file.Path, file.Contextual))
			} else if file.Contextual > 0 {
				report.Violations = append(report.Violations, fmt.Sprintf("%s: expected exclusively proven findings, but got %d contextual finding(s)", file.Path, file.Contextual))
			}
		case "contextual":
			if file.Contextual == 0 {
				report.Violations = append(report.Violations, fmt.Sprintf("%s: expected contextual finding, but got %d proven finding(s)", file.Path, file.Proven))
			}
		default:
			report.Violations = append(report.Violations, fmt.Sprintf("%s: invalid expected-confidence %q (must be 'proven' or 'contextual')", file.Path, file.Expectation.ExpectedConfidence))
		}
	}

	// 4. MissingEvidence validation
	if len(file.Expectation.MissingEvidence) > 0 {
		actualEvMap := make(map[string]bool)
		for _, ev := range file.MissingEvidence {
			actualEvMap[ev] = true
		}
		for _, requiredEv := range file.Expectation.MissingEvidence {
			if !actualEvMap[requiredEv] {
				report.Violations = append(report.Violations, fmt.Sprintf("%s: expected missing-evidence token %q was not emitted; actual missing evidence: %v", file.Path, requiredEv, file.MissingEvidence))
			}
		}
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
