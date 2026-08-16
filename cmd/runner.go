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

	"github.com/schollz/progressbar/v3"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/rules"
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
	if cfg.AnalyzeTests {
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

func runAnalysisPipeline(filesToAnalyze []string, cfg *config.Config) []*rules.Finding {
	allFindings := []*rules.Finding{}

	var wg sync.WaitGroup
	var mu sync.Mutex

	showProgressBar := !verboseFlag

	var bar *progressbar.ProgressBar
	if showProgressBar {
		bar = progressbar.NewOptions(len(filesToAnalyze),
			progressbar.OptionSetDescription("Analyzing files..."),
			progressbar.OptionSetWidth(50),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetPredictTime(true),
			progressbar.OptionSetWriter(os.Stderr),
		)
	} else if !verboseFlag {
		fmt.Fprintf(os.Stderr, "Analyzing %d files...\n", len(filesToAnalyze))
	}

	numCPUs := runtime.NumCPU()
	numWorkers := numCPUs

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
					if bar != nil {
						bar.Add(1)
					}
				}
			}()

			if verboseFlag {
				fmt.Fprintf(os.Stderr, "Analyzing file: %s\n", filePath)
			}

			analysisResult, analyzeErr := analyzerInstance.AnalyzeFile(filePath)

			if analyzeErr != nil {
				if verboseFlag {
					fmt.Fprintf(os.Stderr, "[ERROR] Error analyzing file '%s': %v\n", filePath, analyzeErr)
				}
				if bar != nil {
					bar.Add(1)
				}
				return
			}

			if len(analysisResult.Findings) > 0 {
				localFindings := make([]*rules.Finding, 0, len(analysisResult.Findings))
				for i := range analysisResult.Findings {
					localFindings = append(localFindings, &analysisResult.Findings[i])
				}

				mu.Lock()
				allFindings = append(allFindings, localFindings...)
				mu.Unlock()
			}

			if bar != nil {
				bar.Add(1)
			}
		}(fileToAnalyze)
	}

	wg.Wait()

	dataClumpsAnalyzer := traditional.GetGlobalDataClumpsAnalyzer()
	dataClumpsFindings := dataClumpsAnalyzer.GenerateFindings()
	if dataClumpsFindings != nil {
		mu.Lock()
		allFindings = append(allFindings, dataClumpsFindings...)
		mu.Unlock()
	}

	sort.Slice(allFindings, func(i, j int) bool {
		if allFindings[i].Filepath != allFindings[j].Filepath {
			return allFindings[i].Filepath < allFindings[j].Filepath
		}
		if allFindings[i].Location != nil && allFindings[j].Location != nil {
			if allFindings[i].Location.StartLine != allFindings[j].Location.StartLine {
				return allFindings[i].Location.StartLine < allFindings[j].Location.StartLine
			}
			return allFindings[i].Location.StartColumn < allFindings[j].Location.StartColumn
		}
		if allFindings[i].Location == nil && allFindings[j].Location != nil {
			return true
		}
		if allFindings[i].Location != nil && allFindings[j].Location == nil {
			return false
		}
		return allFindings[i].RuleID < allFindings[j].RuleID
	})

	if showProgressBar {
		fmt.Fprint(os.Stderr, "\n\n")
	} else if !verboseFlag {
		fmt.Fprint(os.Stderr, "\n")
	}

	return allFindings
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
