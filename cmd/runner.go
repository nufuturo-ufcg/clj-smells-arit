package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
	"github.com/thlaurentino/arit/internal/rules/traditional"
)

func resolveConfigDir(filesToAnalyze []string) string {
	configDir := "."
	if len(filesToAnalyze) > 0 {
		firstFileAbs, err := filepath.Abs(filesToAnalyze[0])
		if err == nil {
			parentDir := filepath.Dir(firstFileAbs)

			for parentDir != "/" && parentDir != "." {
				gitPath := filepath.Join(parentDir, ".git")
				modPath := filepath.Join(parentDir, "go.mod")
				projCljPath := filepath.Join(parentDir, "project.clj")
				depsEdnPath := filepath.Join(parentDir, "deps.edn")

				gitInfo, gitErr := os.Stat(gitPath)
				modInfo, modErr := os.Stat(modPath)
				_, projErr := os.Stat(projCljPath)
				_, depsErr := os.Stat(depsEdnPath)

				if (gitErr == nil && gitInfo.IsDir()) || (modErr == nil && !modInfo.IsDir()) || projErr == nil || depsErr == nil {
					configDir = parentDir
					break
				}
				parentDir = filepath.Dir(parentDir)
			}
			if configDir == "." {
				configDir = filepath.Dir(firstFileAbs)
			}
		}
	}
	return configDir
}

func filterTestFiles(filesToAnalyze []string, cfg *config.Config) []string {
	if analyzeTestsFlag || cfg.AnalyzeTests {
		return filesToAnalyze
	}

	var filteredFiles []string
	for _, file := range filesToAnalyze {
		if strings.Contains(file, "/test/") || strings.Contains(file, "/tests/") || strings.HasSuffix(file, "_test.clj") || strings.HasSuffix(file, "-test.clj") {
			continue
		}
		filteredFiles = append(filteredFiles, file)
	}
	return filteredFiles
}

func runAnalysisPipeline(filesToAnalyze []string, cfg *config.Config, discoveryDuration time.Duration) ([]*rules.Finding, rules.SemanticDiagnosticsReport) {
	allFindings := []*rules.Finding{}
	var semanticDiagnostics rules.SemanticDiagnosticsReport
	var timingTotals analyzer.PhaseTimings
	timingTotals.FileDiscovery = discoveryDuration
	var projectIndex *semantics.ProjectIndex
	if expCrossNsFlag {
		projectIndex = semantics.NewProjectIndex()
		indexStart := time.Now()
		for _, fileToAnalyze := range filesToAnalyze {
			if err := projectIndex.IndexFile(fileToAnalyze); err != nil && verboseFlag {
				fmt.Fprintf(os.Stderr, "[WARN] Could not index %s for cross-namespace analysis: %v\n", fileToAnalyze, err)
			}
		}
		projectIndex.BuildMacroCallSites()
		projectIndex.MarkMacroCallSitesComplete()
		if timingFlag {
			timingTotals.ProjectIndex = time.Since(indexStart)
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	if progressEnabled() {
		fmt.Fprintf(os.Stderr, "Arit · analyzing %d files\n", len(filesToAnalyze))
	}
	progress := newProgressReporter(len(filesToAnalyze), os.Stderr, progressEnabled())

	numCPUs := runtime.NumCPU()
	numWorkers := automaticWorkerCount(len(filesToAnalyze), numCPUs)
	if maxWorkersFlag > 0 && maxWorkersFlag < numWorkers {
		numWorkers = maxWorkersFlag
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	if len(filesToAnalyze) > 500 {
		numWorkers = numCPUs * 2
		if numWorkers > 16 {
			numWorkers = 16
		}
	} else if len(filesToAnalyze) > 100 {
		numWorkers = numCPUs + (numCPUs / 2)
		if numWorkers > 12 {
			numWorkers = 12
		}
	} else {
		if numWorkers < 2 {
			numWorkers = 2
		} else if numWorkers > 8 {
			numWorkers = 8
		}
	}

	if len(filesToAnalyze) < numWorkers && len(filesToAnalyze) < 10 {
		numWorkers = len(filesToAnalyze)
	}

	if verboseFlag {
		fmt.Fprintf(os.Stderr, "Using %d workers for %d files (detected %d CPUs)\n", numWorkers, len(filesToAnalyze), numCPUs)
	}

	analyzer.EnableExperimentalMacroExpansion = expMacroExpansionFlag
	analyzer.EnableExperimentalCrossNamespace = expCrossNsFlag
	analyzer.EnableExperimentalTypeInference = expTypeInferenceFlag
	analyzer.EnableExperimentalAsyncCFG = expAsyncCfgFlag
	analyzer.EnableTiming = timingFlag

	semaphore := make(chan struct{}, numWorkers)
	analyzerInstance := analyzer.NewAnalyzer(cfg)

	for _, fileToAnalyze := range filesToAnalyze {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() {
				<-semaphore
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "[PANIC RECOVERED] in goroutine for file '%s': %v\n", filePath, r)
					if verboseFlag {
						fmt.Fprintf(os.Stderr, "Stack trace: %s\n", debug.Stack())
					}
					progress.Add(1)
				}
			}()

			if verboseFlag {
				fmt.Fprintf(os.Stderr, "Analyzing file: %s\n", filePath)
			}

			analysisResult, analyzeErr := analyzerInstance.AnalyzeFileWithProjectIndex(filePath, projectIndex)

			if analyzeErr != nil {
				if verboseFlag {
					fmt.Fprintf(os.Stderr, "[ERROR] Error analyzing file '%s': %v\n", filePath, analyzeErr)
				}
				progress.Add(1)
				return
			}

			mu.Lock()
			semanticDiagnostics.Merge(analysisResult.SemanticDiagnostics)
			mu.Unlock()

			if len(analysisResult.Findings) > 0 {
				localFindings := make([]*rules.Finding, 0, len(analysisResult.Findings))
				for i := range analysisResult.Findings {
					localFindings = append(localFindings, &analysisResult.Findings[i])
				}

				mu.Lock()
				allFindings = append(allFindings, localFindings...)
				mu.Unlock()
			}

			if timingFlag {
				mu.Lock()
				timingTotals.Add(analysisResult.Timings)
				mu.Unlock()
			}

			progress.Add(1)
		}(fileToAnalyze)
	}

	wg.Wait()
	progress.Finish()

	dataClumpsAnalyzer := traditional.GetGlobalDataClumpsAnalyzer()
	dataClumpsFindings := dataClumpsAnalyzer.GenerateFindings()
	if dataClumpsFindings != nil {
		mu.Lock()
		allFindings = append(allFindings, dataClumpsFindings...)
		mu.Unlock()
	}
	if timingFlag {
		fmt.Fprintf(os.Stderr, "Timing by stage (sum across files): discovery=%s index=%s parse=%s rich-tree=%s macro-expansion=%s resolution=%s summaries=%s semantic-facts=%s rules=%s postprocess=%s\n",
			timingTotals.FileDiscovery,
			timingTotals.ProjectIndex,
			timingTotals.Parse,
			timingTotals.BuildRichTree,
			timingTotals.MacroExpansion,
			timingTotals.Resolution,
			timingTotals.FunctionSummaries,
			timingTotals.SemanticFacts,
			timingTotals.RuleTraversal,
			timingTotals.Postprocess,
		)
	}
	for _, finding := range allFindings {
		rules.MarkContextualFinding(finding)
	}

	sort.Slice(allFindings, func(i, j int) bool {
		if allFindings[i].Filepath != allFindings[j].Filepath {
			return allFindings[i].Filepath < allFindings[j].Filepath
		}
		if allFindings[i].Location != nil && allFindings[j].Location != nil {
			if allFindings[i].Location.StartLine != allFindings[j].Location.StartLine {
				return allFindings[i].Location.StartLine < allFindings[j].Location.StartLine
			}
			if allFindings[i].Location.StartColumn != allFindings[j].Location.StartColumn {
				return allFindings[i].Location.StartColumn < allFindings[j].Location.StartColumn
			}
		}
		if allFindings[i].Location == nil && allFindings[j].Location != nil {
			return true
		}
		if allFindings[i].Location != nil && allFindings[j].Location == nil {
			return false
		}
		if allFindings[i].RuleID != allFindings[j].RuleID {
			return allFindings[i].RuleID < allFindings[j].RuleID
		}
		if allFindings[i].Message != allFindings[j].Message {
			return allFindings[i].Message < allFindings[j].Message
		}
		return allFindings[i].ASTFingerprint < allFindings[j].ASTFingerprint
	})

	return allFindings, semanticDiagnostics
}

func automaticWorkerCount(fileCount, cpuCount int) int {
	if cpuCount < 1 {
		cpuCount = 1
	}
	numWorkers := cpuCount
	if fileCount > 500 {
		numWorkers = cpuCount * 2
		if numWorkers > 16 {
			numWorkers = 16
		}
	} else if fileCount > 100 {
		numWorkers = cpuCount + (cpuCount / 2)
		if numWorkers > 12 {
			numWorkers = 12
		}
	} else {
		if numWorkers < 2 {
			numWorkers = 2
		} else if numWorkers > 8 {
			numWorkers = 8
		}
	}
	if fileCount > 0 && fileCount < 10 && fileCount < numWorkers {
		numWorkers = fileCount
	}
	if numWorkers < 1 {
		return 1
	}
	return numWorkers
}

func findClojureFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error accessing path %q: %v\n", path, err)
			return nil
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))

			if ext == ".clj" || ext == ".cljs" || ext == ".cljc" {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the path %q: %w", dir, err)
	}

	sort.Strings(files)

	return files, nil
}
