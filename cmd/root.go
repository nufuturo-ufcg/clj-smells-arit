package cmd

import (
	"fmt"
	io "io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reporter"
	"github.com/thlaurentino/arit/internal/rules"
)

var (
	formatFlag  string
	verboseFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "arit [file-or-dir...]",
	Short: "Arit is a static analyzer for Clojure code.",
	Long: `Arit - Static Analysis for Clojure Code

###############
    • 
┏┓┏┓┓╋
┗┻┛ ┗┗
      
###############

Arit analyzes Clojure files for potential issues,
style violations, and opportunities for improvement.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		fmt.Print(`
###############
    • 
┏┓┏┓┓╋
┗┻┛ ┗┗
      
###############

Arit - Static Analysis for Clojure Code

`)

		filesToAnalyze := []string{}

		for _, arg := range args {
			fileInfo, err := os.Stat(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error accessing argument %q: %v\n", arg, err)
				continue
			}

			if fileInfo.IsDir() {
				cljFiles, err := findClojureFiles(arg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error finding Clojure files in directory %q: %v\n", arg, err)
					continue
				}
				filesToAnalyze = append(filesToAnalyze, cljFiles...)
			} else {
				ext := strings.ToLower(filepath.Ext(arg))
				if ext == ".clj" || ext == ".cljs" || ext == ".cljc" {
					filesToAnalyze = append(filesToAnalyze, arg)
				} else {
					fmt.Fprintf(os.Stderr, "Warning: Skipping non-Clojure file %q\n", arg)
				}
			}
		}

		if len(filesToAnalyze) == 0 {
			fmt.Fprintln(os.Stderr, "No Clojure files found to analyze.")
			return nil
		}

		sort.Strings(filesToAnalyze)

		configDir := "."
		if len(filesToAnalyze) > 0 {
			firstFileAbs, err := filepath.Abs(filesToAnalyze[0])
			if err == nil {
				parentDir := filepath.Dir(firstFileAbs)

				for parentDir != "/" && parentDir != "." {
					gitPath := filepath.Join(parentDir, ".git")
					modPath := filepath.Join(parentDir, "go.mod")
					gitInfo, gitErr := os.Stat(gitPath)
					modInfo, modErr := os.Stat(modPath)
					if (gitErr == nil && gitInfo.IsDir()) || (modErr == nil && !modInfo.IsDir()) {
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

		cfg, err := config.LoadConfig(configDir)
		if err != nil {

			if verboseFlag {
				fmt.Fprintf(os.Stderr, "Warning: Error loading .arit.yaml config from %s: %v. Using default settings.\n", configDir, err)
			}
			cfg = &config.Config{
				EnabledRules: make(map[string]bool),
				RuleConfig:   make(map[string]config.RuleSettings),
			}
		}

		outputFormat := reporter.ReportFormat(formatFlag)
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

		semaphore := make(chan struct{}, numWorkers)

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
					}
				}()

				if verboseFlag {
					fmt.Fprintf(os.Stderr, "Analyzing file: %s\n", filePath)
				}

				done := make(chan bool, 1)
				var analysisResult analyzer.AnalysisResult
				var analyzeErr error

				go func() {
					defer func() {
						if r := recover(); r != nil {
							analyzeErr = fmt.Errorf("panic during analysis: %v", r)
						}
						done <- true
					}()
					analysisResult, analyzeErr = analyzer.AnalyzeFile(filePath, cfg)
				}()

				select {
				case <-done:

				case <-time.After(30 * time.Second):
					analyzeErr = fmt.Errorf("analysis timeout after 30 seconds")
					if verboseFlag {
						fmt.Fprintf(os.Stderr, "[TIMEOUT] Analysis timed out for file '%s'\n", filePath)
					}
				}

				if analyzeErr != nil {
					if verboseFlag {
						fmt.Fprintf(os.Stderr, "[ERROR] Error analyzing file '%s': %v\n", filePath, analyzeErr)
					}
					return
				}

				var localFindings []*rules.Finding
				if analysisResult.Findings != nil {
					localFindings = make([]*rules.Finding, 0, len(analysisResult.Findings))
					for _, finding := range analysisResult.Findings {
						findingCopy := &rules.Finding{
							RuleID:   finding.RuleID,
							Message:  finding.Message,
							Filepath: finding.Filepath,
							Location: finding.Location,
							Severity: finding.Severity,
						}
						localFindings = append(localFindings, findingCopy)
					}
				}

				if len(localFindings) > 0 {
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

		dataClumpsAnalyzer := rules.GetGlobalDataClumpsAnalyzer()
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

		if outputFormat != reporter.FormatSummary {
			switch outputFormat {
			case reporter.FormatJSON:
				fmt.Fprintf(os.Stderr, "Report generated in JSON format.\n")
			case reporter.FormatHTML:
				fmt.Fprintf(os.Stderr, "Report generated in HTML format.\n")
			case reporter.FormatMarkdown:
				fmt.Fprintf(os.Stderr, "Report generated in Markdown format.\n")
			case reporter.FormatText:
				fmt.Fprintf(os.Stderr, "Report generated in text format.\n")
			default:
				fmt.Fprintf(os.Stderr, "Report generated in %s format.\n", outputFormat)
			}
		}

		rep := reporter.NewReporter(outputFormat)
		if rep == nil {
			return fmt.Errorf("unsupported report format: %s", outputFormat)
		}

		var outputWriter io.Writer = os.Stdout
		err = rep.Report(allFindings, outputWriter)
		if err != nil {
			return fmt.Errorf("error generating report: %w", err)
		}

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&formatFlag, "format", "f", "summary", "Output format (summary, text, json, html, markdown)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	rootCmd.AddCommand(listRulesCmd)
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

var _ = rules.Rule{}
