package cmd

import (
	"fmt"
	io "io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reporter"
	"github.com/thlaurentino/arit/internal/rules"
)

var (
	formatFlag string
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
			fmt.Fprintf(os.Stderr, "Warning: Error loading .arit.yaml config from %s: %v. Using default settings.\n", configDir, err)

			cfg = &config.Config{
				EnabledRules: make(map[string]bool),
				RuleConfig:   make(map[string]config.RuleSettings),
			}
		}

		outputFormat := reporter.ReportFormat(formatFlag)
		allFindings := []*rules.Finding{}

		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, fileToAnalyze := range filesToAnalyze {
			wg.Add(1)
			go func(filePath string) {
				defer wg.Done()

				defer func() {
					if r := recover(); r != nil {
						log.Printf("[PANIC RECOVERED] in goroutine for file '%s': %v", filePath, r)
					}
				}()

				log.Printf("Analyzing file: %s", filePath)
				analysisResult, analyzeErr := analyzer.AnalyzeFile(filePath, cfg)
				if analyzeErr != nil {
					log.Printf("[ERROR Main Goroutine] Error analyzing file '%s': %v", filePath, analyzeErr)
					return
				}

				mu.Lock()
				if analysisResult.Findings != nil {
					for _, finding := range analysisResult.Findings {

						findingCopy := rules.Finding{
							RuleID:   finding.RuleID,
							Message:  finding.Message,
							Filepath: finding.Filepath,
							Location: finding.Location,
							Severity: finding.Severity,
						}
						allFindings = append(allFindings, &findingCopy)
					}
				}
				mu.Unlock()
			}(fileToAnalyze)
		}

		wg.Wait()

		fmt.Fprintf(os.Stderr, "\n--- Analysis Findings (%d) ---\n", len(allFindings))
		rep, err := reporter.NewReporter(outputFormat)
		if err != nil {
			return fmt.Errorf("error creating reporter: %w", err)
		}

		var outputWriter io.Writer = os.Stdout
		err = rep.Report(allFindings, outputWriter)
		if err != nil {
			return fmt.Errorf("error generating report: %w", err)
		}

		fmt.Fprintln(os.Stderr, "\nAnalysis complete.")
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&formatFlag, "format", "f", string(reporter.FormatText), "Output format (text, json, html, markdown, sarif)")
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

	return files, nil
}

var _ = rules.Rule{}
